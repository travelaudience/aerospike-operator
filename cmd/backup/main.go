/*
Copyright 2018 The aerospike-operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

const (
	compress = true
)

var (
	backupFS   *flag.FlagSet
	restoreFS  *flag.FlagSet
	debug      bool
	bucketName string
	name       string
	secretPath string
	pipePath   string
	ctx        context.Context
)

func init() {
	backupFS = flag.NewFlagSet("backup", flag.ExitOnError)
	backupFS.BoolVar(&debug, "debug", false, "whether to enable debug logging")
	backupFS.StringVar(&bucketName, "bucket-name", "", "the name of the bucket to upload/download backup to/from")
	backupFS.StringVar(&name, "name", "", "the name of the backup file to be stored on GCS")
	backupFS.StringVar(&secretPath, "secret-path", "/creds/key.json", "the path of the key.json file to use as bucket credentials")
	backupFS.StringVar(&pipePath, "pipe-path", "/shared/pipe.tmp", "the path of the named pipe used to transfer data between containers")

	restoreFS = flag.NewFlagSet("restore", flag.ExitOnError)
	restoreFS.BoolVar(&debug, "debug", false, "whether to enable debug logging")
	restoreFS.StringVar(&bucketName, "bucket-name", "", "the name of the bucket to upload/download backup to/from")
	restoreFS.StringVar(&name, "name", "", "the name of the backup file to be stored on GCS")
	restoreFS.StringVar(&secretPath, "secret-path", "/creds/key.json", "the path of the key.json file to use as bucket credentials")
	restoreFS.StringVar(&pipePath, "pipe-path", "/data/pipe.tmp", "the path of the named pipe used to transfer data between containers")
}

func printUsage() {
	fmt.Printf("\nusage: backup <command> [<args>]\n\n")
	fmt.Println("Available commands: ")
	fmt.Printf("\tsave\t\tSave backup to bucket\n")
	fmt.Printf("\trestore\t\tRestore backup from bucket\n")
	fmt.Println()
}

func parseCommandArgs() {
	if len(os.Args) == 1 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "save":
		backupFS.Parse(os.Args[2:])
	case "restore":
		restoreFS.Parse(os.Args[2:])
	default:
		fmt.Printf("\n%q is not valid command.\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	if bucketName == "" || name == "" {
		if backupFS.Parsed() {
			backupFS.PrintDefaults()
		} else if restoreFS.Parsed() {
			restoreFS.PrintDefaults()
		}
		os.Exit(2)
	}
}

func main() {
	parseCommandArgs()
	ctx = context.Background()

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(secretPath))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)
	if _, err = bucket.Attrs(ctx); err != nil {
		log.Fatal(err)
	}
	backupObject := bucket.Object(name)
	metaObject := bucket.Object(fmt.Sprintf("%s.meta", name))

	var bytesTransferred int64
	if backupFS.Parsed() {
		if err := writeMetadata(metaObject); err != nil {
			log.Fatal(err)
		}
		if bytesTransferred, err = backup(backupObject); err != nil {
			log.Fatal(err)
		}
	} else if restoreFS.Parsed() {
		if err := readMetadata(metaObject); err != nil {
			log.Fatal(err)
		}

		// Let the destination read EOF and close the pipe
		// (ensures metadata and data are not sent together)
		time.Sleep(500 * time.Millisecond)

		if bytesTransferred, err = restore(backupObject); err != nil {
			log.Fatal(err)
		}
	}
	log.Infof("Backup size: %d bytes", bytesTransferred)
}

func backup(backupObject *storage.ObjectHandle) (n int64, err error) {
	pipe, err := os.OpenFile(pipePath, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		log.Fatal("Open named pipe error:", err)
	}
	defer os.Remove(pipePath)
	defer pipe.Close()

	var reader io.Reader
	if debug {
		reader = NewReaderWithProgress(pipe)
	} else {
		reader = pipe
	}
	n, err = transferToGCS(reader, backupObject)
	return
}

func restore(obj *storage.ObjectHandle) (n int64, err error) {
	pipe, err := os.OpenFile(pipePath, os.O_CREATE|os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		log.Fatal("Open named pipe error:", err)
	}
	defer os.Remove(pipePath)
	defer pipe.Close()

	var writer io.Writer
	if debug {
		writer = NewWriterWithProgress(pipe)
	} else {
		writer = pipe
	}
	n, err = transferFromGCS(writer, obj)
	return
}

func transferToGCS(r io.Reader, obj *storage.ObjectHandle) (n int64, err error) {
	w := obj.NewWriter(ctx)
	defer w.Close()

	if compress {
		gz := gzip.NewWriter(w)
		defer gz.Close()
		n, err = io.Copy(gz, r)
	} else {
		n, err = io.Copy(w, r)
	}
	return
}

func transferFromGCS(w io.Writer, obj *storage.ObjectHandle) (n int64, err error) {
	r, err := obj.NewReader(ctx)
	if err != nil {
		return
	}
	defer r.Close()

	if compress {
		gz, err := gzip.NewReader(r)
		if err != nil {
			return 0, err
		}
		defer gz.Close()
		n, err = io.Copy(w, gz)
	} else {
		n, err = io.Copy(w, r)
	}
	return
}

func writeMetadata(metaObject *storage.ObjectHandle) error {
	pipe, err := os.OpenFile(pipePath, os.O_CREATE|os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		log.Fatal("Open named pipe error:", err)
	}
	defer pipe.Close()

	w := metaObject.NewWriter(ctx)
	defer w.Close()

	_, err = io.Copy(w, pipe)
	return err
}

func readMetadata(metaObject *storage.ObjectHandle) error {
	r, err := metaObject.NewReader(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	pipe, err := os.OpenFile(pipePath, os.O_CREATE|os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		log.Fatal("Open named pipe error:", err)
	}
	defer pipe.Close()

	_, err = io.Copy(pipe, r)
	return err
}
