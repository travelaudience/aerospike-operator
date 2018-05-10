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
	"io"
	"os"
	"os/exec"
	"strconv"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

var (
	fs              *flag.FlagSet
	debug           bool
	targetHost      string
	targetNamespace string
	targetPort      int
	bucket          string
	name            string
	compress        bool
	backupTask      bool
	restoreTask     bool
	secretPath      string
	ctx             context.Context
)

func init() {
	fs = flag.NewFlagSet("", flag.ExitOnError)
	fs.BoolVar(&debug, "debug", false, "whether to enable debug logging")
	fs.StringVar(&targetHost, "host", "localhost", "the host of the target aerospike cluster")
	fs.StringVar(&targetNamespace, "namespace", "", "the namespace to be backed up")
	fs.IntVar(&targetPort, "port", 3000, "the port of the target aerospike cluster")
	fs.StringVar(&bucket, "bucket", "", "the bucket to upload/download backup to/from")
	fs.StringVar(&name, "name", "", "the name of the backup file to be stored on GCS")
	fs.BoolVar(&backupTask, "backup", false, "run backup task")
	fs.BoolVar(&restoreTask, "restore", false, "run restore task")
	fs.BoolVar(&compress, "compress", false, "use compressed backup/restore files (gzip)")
	fs.StringVar(&secretPath, "secret-path", "/creds/key.json", "the host of the target aerospike cluster")
	fs.Parse(os.Args[1:])
	ctx = context.Background()
}

func validateArgs() {
	if backupTask || restoreTask {
		return
	}
	if bucket != "" && name != "" && targetNamespace != "" {
		return
	}
	fs.PrintDefaults()
	os.Exit(1)
}

func main() {
	validateArgs()
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(secretPath))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	bh := client.Bucket(bucket)
	// Check if the bucket exists
	if _, err = bh.Attrs(ctx); err != nil {
		log.Fatal(err)
	}
	obj := bh.Object(name)

	var bytesTransfered int64
	if backupTask {
		bytesTransfered, err = backup(obj)
	} else if restoreTask {
		bytesTransfered, err = restore(obj)
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Backup size: %d bytes", bytesTransfered)
}

func backup(obj *storage.ObjectHandle) (n int64, err error) {
	var reader io.Reader

	cmd := exec.Command("asbackup",
		"-h", targetHost,
		"-p", strconv.Itoa(targetPort),
		"-n", targetNamespace,
		"--output-file", "-")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	if debug {
		reader = NewReaderWithProgress(stdout)
	} else {
		reader = stdout
	}

	nBytesChan := make(chan int64, 1)
	errorChan := make(chan error, 1)
	go func() {
		defer close(errorChan)
		defer close(nBytesChan)
		defer stdout.Close()
		n, err := transferToGCS(reader, obj)
		if err != nil {
			errorChan <- err
		} else {
			nBytesChan <- n
		}
	}()

	err = cmd.Run()
	if err != nil {
		return
	}
	n = <-nBytesChan
	err = <-errorChan
	return
}

func restore(obj *storage.ObjectHandle) (n int64, err error) {
	var writer io.Writer

	cmd := exec.Command("asrestore",
		"-h", targetHost,
		"-p", strconv.Itoa(targetPort),
		"--input-file", "-")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return
	}

	if debug {
		writer = NewWriterWithProgress(stdin)
	} else {
		writer = stdin
	}

	nBytesChan := make(chan int64, 1)
	errorChan := make(chan error, 1)
	go func() {
		defer close(errorChan)
		defer close(nBytesChan)
		defer stdin.Close()
		n, err := transferFromGCS(writer, obj)
		if err != nil {
			errorChan <- err
		} else {
			nBytesChan <- n
		}
	}()

	err = cmd.Run()
	if err != nil {
		return
	}
	n = <-nBytesChan
	err = <-errorChan
	return
}

func transferToGCS(r io.Reader, obj *storage.ObjectHandle) (n int64, err error) {
	w := obj.NewWriter(ctx)
	defer w.Close()

	if compress {
		gz := gzip.NewWriter(w)
		defer gz.Close()
		if n, err = io.Copy(gz, r); err != nil {
			return
		}
	} else {
		if n, err = io.Copy(w, r); err != nil {
			return
		}
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
		if n, err = io.Copy(w, gz); err != nil {
			return 0, err
		}
	} else {
		if n, err = io.Copy(w, r); err != nil {
			return
		}
	}
	return
}
