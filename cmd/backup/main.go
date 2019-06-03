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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	"github.com/travelaudience/aerospike-operator/internal/backuprestore"
	"github.com/travelaudience/aerospike-operator/internal/backuprestore/gcs"
	flagutils "github.com/travelaudience/aerospike-operator/internal/utils/flags"
)

const (
	backupCommand  = "backup"
	restoreCommand = "restore"

	debugFlag      = "debug"
	bucketNameFlag = "bucket-name"
	nameFlag       = "name"
	secretPathFlag = "secret-path"
	hostFlag       = "host"
	portFlag       = "port"
	namespaceFlag  = "namespace"
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
	bfs.BoolVar(&debug, debugFlag, false, "[DEPRECATED] whether to enable debug logging")
	bfs.StringVar(&bucketName, bucketNameFlag, "", "the name of the bucket to upload the backup to")
	bfs.StringVar(&name, nameFlag, "", "the name of the backup file to be stored on GCS")
	bfs.StringVar(&secretPath, secretPathFlag, "/secret/key.json", "the path to the service account credentials file")
	bfs.StringVar(&host, hostFlag, "", "the host to which asbackup will connect")
	bfs.IntVar(&port, portFlag, 3000, "the port to which asbackup will connect")
	bfs.StringVar(&namespace, namespaceFlag, "", "the name of the namespace which to backup")

	rfs = flag.NewFlagSet(restoreCommand, flag.ExitOnError)
	rfs.BoolVar(&debug, debugFlag, false, "[DEPRECATED] whether to enable debug logging")
	rfs.StringVar(&bucketName, bucketNameFlag, "", "the name of the bucket to download the backup from")
	rfs.StringVar(&name, nameFlag, "", "the name of the backup file to be retrieved from GCS")
	rfs.StringVar(&secretPath, secretPathFlag, "/secret/key.json", "the path to the service account credentials file")
	rfs.StringVar(&host, hostFlag, "", "the host to which asrestore will connect")
	rfs.IntVar(&port, portFlag, 3000, "the port to which asrestore will connect")
	rfs.StringVar(&namespace, namespaceFlag, "", "the name of the namespace which to restore data into")
}

func main() {
	if len(os.Args) == 1 {
		log.Fatalf("too few arguments")
	}
	switch os.Args[1] {
	case backupCommand:
		bfs.Parse(os.Args[2:])

		// warn about deprecated flags
		flagutils.DeprecateFlags(bfs, debugFlag)

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

		// warn about deprecated flags
		flagutils.DeprecateFlags(rfs, debugFlag)

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
	client, err := gcs.NewGCSClientFromCredentials(secretPath)
	if err != nil {
		return err
	}
	defer client.Close()

	// dump metadata to the meta file
	log.Debug("dumping metadata")
	if err := dumpMetadata(client); err != nil {
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
	if err := client.TransferToGCS(o, bucketName, backuprestore.GetBackupObjectName(name)); err != nil {
		return err
	}
	// wait for asbackup to terminate
	return cmd.Wait()
}

// doRestore performs a restore operation to the target namespace.
func doRestore() error {
	// initialize the gcs client and get handles to the meta and backup objects
	log.Debug("initing cloud storage")
	client, err := gcs.NewGCSClientFromCredentials(secretPath)
	if err != nil {
		return err
	}
	defer client.Close()

	// read metadata to the meta file
	log.Debug("reading metadata")
	n, err := readMetadata(client)
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
	if err := client.TransferFromGCS(i, bucketName, backuprestore.GetBackupObjectName(name)); err != nil {
		return err
	}
	// close stdin when we're done
	if err := i.Close(); err != nil {
		return err
	}
	// wait for asrestore to terminate
	return cmd.Wait()
}

// dumpMetadata dumps backup metadata to GCS.
func dumpMetadata(client *gcs.GCSClient) error {
	// get the object
	metaObject, err := client.GetObject(bucketName, backuprestore.GetMetadataObjectName(name))
	if err != nil {
		return err
	}
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
func readMetadata(client *gcs.GCSClient) (*backupMetadata, error) {
	// get the object
	metaObject, err := client.GetObject(bucketName, backuprestore.GetMetadataObjectName(name))
	if err != nil {
		return nil, err
	}
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
