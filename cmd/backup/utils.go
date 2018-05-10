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
	"io"

	log "github.com/sirupsen/logrus"
)

type ReaderWithProgress struct {
	io.Reader
	total int64
}

func NewReaderWithProgress(r io.Reader) *ReaderWithProgress {
	return &ReaderWithProgress{r, 0}
}

func (rwp *ReaderWithProgress) Read(p []byte) (int, error) {
	n, err := rwp.Reader.Read(p)
	rwp.total += int64(n)
	if err == nil {
		log.Debugf("Read %d bytes for a total of %d", n, rwp.total)
	}
	return n, err
}

type WriterWithProgress struct {
	io.Writer
	total int64
}

func NewWriterWithProgress(r io.Writer) *WriterWithProgress {
	return &WriterWithProgress{r, 0}
}

func (wwp *WriterWithProgress) Write(p []byte) (int, error) {
	n, err := wwp.Writer.Write(p)
	wwp.total += int64(n)
	if err == nil {
		log.Debugf("Wrote %d bytes for a total of %d", n, wwp.total)
	}
	return n, err
}
