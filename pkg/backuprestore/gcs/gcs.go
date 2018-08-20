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

package gcs

import (
	"compress/gzip"
	"io"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

type GCSClient struct {
	client *storage.Client
}

// NewGCSClientFromCredentials returns a new GCSClient loading the credentials
// from the file present in the specified path
func NewGCSClientFromCredentials(credentialsFilePath string) (*GCSClient, error) {
	client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(credentialsFilePath))
	if err != nil {
		return nil, err
	}
	return &GCSClient{
		client: client,
	}, nil
}

// NewGCSClientFromJSON returns a new GCSClient loading the
// credentials from the given JSON string
func NewGCSClientFromJSON(jsonBytes []byte) (*GCSClient, error) {
	ctx := context.Background()
	creds, err := google.CredentialsFromJSON(ctx, jsonBytes, storage.ScopeReadWrite)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	return &GCSClient{
		client: client,
	}, nil
}

// Close closes the GCS client contained in the GCSClient
func (h *GCSClient) Close() error {
	return h.client.Close()
}

// GetObject returns a object handle
func (h *GCSClient) GetObject(bucketName, objectName string) (*storage.ObjectHandle, error) {
	// get a handle to the target bucket
	bucket := h.client.Bucket(bucketName)

	// attempt to read the bucket metadata in
	// order to know the bucket exists
	if _, err := bucket.Attrs(context.Background()); err != nil {
		return nil, err
	}

	// delete the object
	return bucket.Object(objectName), nil
}

// DeleteObject deletes the content of the specified object
func (h *GCSClient) DeleteObject(bucketName, objectName string) error {
	// get the object
	obj, err := h.GetObject(bucketName, objectName)
	if err != nil {
		return err
	}
	// delete the object
	return obj.Delete(context.Background())
}

// TransferToGCS streams backup data to GCS.
func (h *GCSClient) TransferToGCS(r io.Reader, bucketName, objectName string) error {
	// get the object
	obj, err := h.GetObject(bucketName, objectName)
	if err != nil {
		return err
	}
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

// TransferFromGCS streams backup data from GCS.
func (h *GCSClient) TransferFromGCS(w io.Writer, bucketName, objectName string) error {
	// get the object
	obj, err := h.GetObject(bucketName, objectName)
	if err != nil {
		return err
	}
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
