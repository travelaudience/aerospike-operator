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

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

var (
	fs          *flag.FlagSet
	debug       bool
	bucket      string
	name        string
	compress    bool
	backupTask  bool
	restoreTask bool
	secretPath  string
	pipePath    string
	ctx         context.Context
)

func init() {
	fs = flag.NewFlagSet("", flag.ExitOnError)
	fs.BoolVar(&debug, "debug", false, "whether to enable debug logging")
	fs.StringVar(&bucket, "bucket", "", "the bucket to upload/download backup to/from")
	fs.StringVar(&name, "name", "", "the name of the backup file to be stored on GCS")
	fs.BoolVar(&backupTask, "backup", false, "run backup task")
	fs.BoolVar(&restoreTask, "restore", false, "run restore task")
	fs.BoolVar(&compress, "compress", false, "use compressed backup/restore files (gzip)")
	fs.StringVar(&secretPath, "secret-path", "/creds/key.json", "the host of the target aerospike cluster")
	fs.StringVar(&pipePath, "pipe-path", "/shared/pipe.tmp", "the path of the named pipe used to transfer data between containers")
	fs.Parse(os.Args[1:])
	ctx = context.Background()
}

func validateArgs() {
	if backupTask || restoreTask {
		return
	}
	if bucket != "" && name != "" {
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
	if _, err = bh.Attrs(ctx); err != nil {
		log.Fatal(err)
	}
	backupObject := bh.Object(name)

	var bytesTransfered int64
	if backupTask {
		bytesTransfered, err = backup(backupObject)
	} else if restoreTask {
		bytesTransfered, err = restore(backupObject)
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Backup size: %d bytes", bytesTransfered)
}

func backup(backupObject *storage.ObjectHandle) (n int64, err error) {
	pipe, err := os.OpenFile(pipePath, os.O_CREATE, os.ModeNamedPipe)
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
	pipe, err := os.OpenFile(pipePath, os.O_RDWR, os.ModeNamedPipe)
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
