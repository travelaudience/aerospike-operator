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

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

const (
	backupSaveCommand    = "save"
	backupRestoreCommand = "restore"
)

var (
	backupFS     *flag.FlagSet
	restoreFS    *flag.FlagSet
	debug        bool
	bucketName   string
	name         string
	secretPath   string
	dataPipePath string
	metaPipePath string
	ctx          context.Context
)

func init() {
	backupFS = flag.NewFlagSet(backupSaveCommand, flag.ExitOnError)
	backupFS.BoolVar(&debug, "debug", false, "whether to enable debug logging")
	backupFS.StringVar(&bucketName, "bucket-name", "", "the name of the bucket to upload the backup to")
	backupFS.StringVar(&name, "name", "", "the name of the backup file to be stored on GCS")
	backupFS.StringVar(&secretPath, "secret-path", "/creds/key.json", "the path to the service account credentials file")
	backupFS.StringVar(&dataPipePath, "data-pipe-path", "/data/data.tmp", "the path to the named pipe used to transfer data between containers")
	backupFS.StringVar(&metaPipePath, "meta-pipe-path", "/data/meta.tmp", "the path to the named pipe used to transfer metadata between containers")

	restoreFS = flag.NewFlagSet(backupRestoreCommand, flag.ExitOnError)
	restoreFS.BoolVar(&debug, "debug", false, "whether to enable debug logging")
	restoreFS.StringVar(&bucketName, "bucket-name", "", "the name of the bucket to download the backup from")
	restoreFS.StringVar(&name, "name", "", "the name of the backup file to be retrieved from GCS")
	restoreFS.StringVar(&secretPath, "secret-path", "/creds/key.json", "the path to the service account credentials file")
	restoreFS.StringVar(&dataPipePath, "data-pipe-path", "/data/data.tmp", "the path to the named pipe used to transfer data between containers")
	restoreFS.StringVar(&metaPipePath, "meta-pipe-path", "/data/meta.tmp", "the path to the named pipe used to transfer metadata between containers")
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
		if _, err := writeMetadata(metaObject); err != nil {
			log.Fatal(err)
		}
		if bytesTransferred, err = backup(backupObject); err != nil {
			log.Fatal(err)
		}
	} else if restoreFS.Parsed() {
		if _, err := readMetadata(metaObject); err != nil {
			log.Fatal(err)
		}
		if bytesTransferred, err = restore(backupObject); err != nil {
			log.Fatal(err)
		}
	}
	log.Infof("backup size: %d bytes", bytesTransferred)
}

func printUsage() {
	fmt.Printf("\nusage: backup <command> [<args>]\n\n")
	fmt.Println("available commands:")
	fmt.Printf("\tsave\t\tsave backup to bucket\n")
	fmt.Printf("\trestore\t\trestore backup from bucket\n")
	fmt.Println()
}

func parseCommandArgs() {
	if len(os.Args) == 1 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case backupSaveCommand:
		backupFS.Parse(os.Args[2:])
	case backupRestoreCommand:
		restoreFS.Parse(os.Args[2:])
	default:
		fmt.Printf("\n%q is not a valid command\n", os.Args[1])
		printUsage()
		os.Exit(1)
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
		os.Exit(1)
	}
}

func backup(backupObject *storage.ObjectHandle) (int64, error) {
	pipe, err := os.OpenFile(dataPipePath, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return 0, err
	}
	defer os.Remove(dataPipePath)
	defer pipe.Close()

	return transferToGCS(pipe, backupObject)
}

func restore(obj *storage.ObjectHandle) (int64, error) {
	pipe, err := os.OpenFile(dataPipePath, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		return 0, err
	}
	defer os.Remove(dataPipePath)
	defer pipe.Close()

	return transferFromGCS(pipe, obj)
}

func transferToGCS(r io.Reader, obj *storage.ObjectHandle) (int64, error) {
	w := obj.NewWriter(ctx)
	defer w.Close()

	gz := gzip.NewWriter(w)
	defer gz.Close()

	return io.Copy(gz, r)
}

func transferFromGCS(w io.Writer, obj *storage.ObjectHandle) (int64, error) {
	r, err := obj.NewReader(ctx)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	gz, err := gzip.NewReader(r)
	if err != nil {
		return 0, err
	}
	defer gz.Close()

	return io.Copy(w, gz)
}

func writeMetadata(metaObject *storage.ObjectHandle) (int64, error) {
	pipe, err := os.OpenFile(metaPipePath, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return 0, err
	}
	defer os.Remove(metaPipePath)
	defer pipe.Close()

	w := metaObject.NewWriter(ctx)
	defer w.Close()

	return io.Copy(w, pipe)
}

func readMetadata(metaObject *storage.ObjectHandle) (int64, error) {
	r, err := metaObject.NewReader(ctx)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	pipe, err := os.OpenFile(metaPipePath, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		return 0, err
	}
	defer os.Remove(metaPipePath)
	defer pipe.Close()

	return io.Copy(pipe, r)
}
