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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

const (
	backupCommand  = "backup"
	restoreCommand = "restore"
)

var (
	bfs *flag.FlagSet
	rfs *flag.FlagSet

	debug      bool
	bucketName string
	name       string
	secretPath string
	host       string
	port       int
	namespace  string
)

// backupMetadata stores metadata about a backup operation.
type backupMetadata struct {
	// Namespace holds the original name of the namespace at the time the backup
	// was performed.
	Namespace string `json:"namespace"`
}

func init() {
	bfs = flag.NewFlagSet(backupCommand, flag.ExitOnError)
	bfs.BoolVar(&debug, "debug", false, "whether to enable debug logging")
	bfs.StringVar(&bucketName, "bucket-name", "", "the name of the bucket to upload the backup to")
	bfs.StringVar(&name, "name", "", "the name of the backup file to be stored on GCS")
	bfs.StringVar(&secretPath, "secret-path", "/secret/key.json", "the path to the service account credentials file")
	bfs.StringVar(&host, "host", "", "the host to which asbackup will connect")
	bfs.IntVar(&port, "port", 3000, "the port to which asbackup will connect")
	bfs.StringVar(&namespace, "namespace", "", "the name of the namespace which to backup")

	rfs = flag.NewFlagSet(restoreCommand, flag.ExitOnError)
	rfs.BoolVar(&debug, "debug", false, "whether to enable debug logging")
	rfs.StringVar(&bucketName, "bucket-name", "", "the name of the bucket to download the backup from")
	rfs.StringVar(&name, "name", "", "the name of the backup file to be retrieved from GCS")
	rfs.StringVar(&secretPath, "secret-path", "/secret/key.json", "the path to the service account credentials file")
	rfs.StringVar(&host, "host", "", "the host to which asrestore will connect")
	rfs.IntVar(&port, "port", 3000, "the port to which asrestore will connect")
	rfs.StringVar(&namespace, "namespace", "", "the name of the namespace which to restore data into")
}

func main() {
	if len(os.Args) == 1 {
		log.Fatalf("too few arguments")
	}
	switch os.Args[1] {
	case backupCommand:
		bfs.Parse(os.Args[2:])
		if debug {
			log.SetLevel(log.DebugLevel)
		}
		log.Info("backup is starting")
		if err := doBackup(); err != nil {
			log.Fatal(err)
		}
		log.Info("backup is complete")
	case restoreCommand:
		rfs.Parse(os.Args[2:])
		if debug {
			log.SetLevel(log.DebugLevel)
		}
		log.Info("restore is starting")
		if err := doRestore(); err != nil {
			log.Fatal(err)
		}
		log.Info("restore is complete")
	default:
		log.Fatalf("invalid command %q", os.Args[1])
	}
}

// doBackup performs a backup operation on the target namespace.
func doBackup() error {
	// initialize the gcs client and get handles to the meta and backup objects
	log.Debug("initing cloud storage")
	client, metaObject, backupObject, err := initGCSObjects()
	if err != nil {
		return err
	}
	defer client.Close()

	// dump metadata to the meta file
	log.Debug("dumping metadata")
	if err := dumpMetadata(metaObject); err != nil {
		return err
	}

	// build the asbackup command
	cmd := exec.Command("asbackup", "-h", host, "-p", strconv.Itoa(port), "-n", namespace, "-o", "-", "-c", "-v")
	// get a handle to stdout
	o, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	// capture asbackup's stderr
	errw := log.New().Writer()
	defer errw.Close()
	cmd.Stderr = errw

	// give some feedback about what is going to be executed
	log.Debug("==== asbackup ====")
	log.Debug(strings.Join(cmd.Args, " "))
	log.Debug("==================")

	// launch the asbackup process
	log.Debug("running asbackup and streaming to cloud storage")
	if err := cmd.Start(); err != nil {
		return err
	}
	// transfer data from asbackup's stdout to gcs
	if err := transferToGCS(o, backupObject); err != nil {
		return err
	}
	// wait for asbackup to terminate
	return cmd.Wait()
}

// doRestore performs a restore operation to the target namespace.
func doRestore() error {
	// initialize the gcs client and get handles to the meta and backup objects
	log.Debug("initing cloud storage")
	client, metaObject, backupObject, err := initGCSObjects()
	if err != nil {
		return err
	}
	defer client.Close()

	// read metadata to the meta file
	log.Debug("reading metadata")
	n, err := readMetadata(metaObject)
	if err != nil {
		return err
	}

	// build the asrestore command
	cmd := exec.Command("asrestore", "-h", host, "-p", strconv.Itoa(port), "-i", "-", "-n", fmt.Sprintf("%s,%s", n.Namespace, namespace), "-v")
	// get a handle to stdin
	i, err := cmd.StdinPipe()
	// capture asrestore's stderr
	errw := log.New().Writer()
	defer errw.Close()
	cmd.Stderr = errw

	// give some feedback about what is going to be executed
	log.Debug("==== asrestore ====")
	log.Debug(strings.Join(cmd.Args, " "))
	log.Debug("===================")

	// launch the asrestore process
	log.Debug("running asrestore and streaming from cloud storage")
	if err := cmd.Start(); err != nil {
		return err
	}
	// transfer data from gcs to asrestore's stdin
	if err := transferFromGCS(i, backupObject); err != nil {
		return err
	}
	// close stdin when we're done
	if err := i.Close(); err != nil {
		return err
	}
	// wait for asrestore to terminate
	return cmd.Wait()
}

// initGCSObjects initializes the GCS client and returns handles to the meta and backup objects.
func initGCSObjects() (*storage.Client, *storage.ObjectHandle, *storage.ObjectHandle, error) {
	// create a gcs client
	client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(secretPath))
	if err != nil {
		return nil, nil, nil, err
	}
	// get a handle to the target bucket
	bucket := client.Bucket(bucketName)
	// attempt to read the bucket metadata
	if _, err = bucket.Attrs(context.Background()); err != nil {
		return nil, nil, nil, err
	}
	// get a handle to the metadata object
	metaObject := bucket.Object(fmt.Sprintf("%s.json", name))
	// get a handle to the backup object
	backupObject := bucket.Object(fmt.Sprintf("%s.asb.gz", name))
	// return the gcs client and the handles to the metadata and backup files
	return client, metaObject, backupObject, nil
}

// transferToGCS streams backup data to GCS.
func transferToGCS(r io.Reader, obj *storage.ObjectHandle) error {
	// create a writer that writes to the target object
	w := obj.NewWriter(context.Background())
	defer w.Close()
	// create a writer that gzips the backup data
	gz := gzip.NewWriter(w)
	defer gz.Close()
	// copy the gziped backup data to the bucket
	if s, err := io.Copy(gz, r); err != nil {
		return err
	} else {
		log.Infof("%d bytes written", s)
		return nil
	}
}

// transferFromGCS streams backup data from GCS.
func transferFromGCS(w io.Writer, obj *storage.ObjectHandle) error {
	// create a reader that reads from the source object
	r, err := obj.NewReader(context.Background())
	if err != nil {
		return err
	}
	defer r.Close()
	// create a reader that ungzips the backup data
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()
	// read the gziped backup data from the bucket
	if r, err := io.Copy(w, gz); err != nil {
		return err
	} else {
		log.Infof("%d bytes read", r)
		return nil
	}
}

// dumpMetadata dumps backup metadata to GCS.
func dumpMetadata(metaObject *storage.ObjectHandle) error {
	// create a writer that writes to the target object
	w := metaObject.NewWriter(context.Background())
	defer w.Close()
	// dump the backup metadata to the writer
	m := &backupMetadata{Namespace: namespace}
	if err := json.NewEncoder(w).Encode(m); err != nil {
		return err
	}
	return nil
}

// readMetadata reads backup metadata from GCS.
func readMetadata(metaObject *storage.ObjectHandle) (*backupMetadata, error) {
	// create a reader that reads from the source object
	r, err := metaObject.NewReader(context.Background())
	if err != nil {
		return nil, err
	}
	defer r.Close()
	// read the backup metadata from the reader
	m := &backupMetadata{}
	if err := json.NewDecoder(r).Decode(m); err != nil {
		return nil, err
	}
	return m, nil
}
