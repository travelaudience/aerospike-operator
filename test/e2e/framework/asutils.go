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

package framework

import (
	"fmt"

	as "github.com/aerospike/aerospike-client-go"
)

type AerospikeClient struct {
	client *as.Client
}

func NewAerospikeClient(host string, port int) (*AerospikeClient, error) {
	c, err := as.NewClientWithPolicy(as.NewClientPolicy(), host, port)
	if err != nil {
		return nil, err
	}
	return &AerospikeClient{client: c}, nil
}

func (ac *AerospikeClient) Close() {
	ac.client.Close()
}

func (ac *AerospikeClient) WriteSequentialIntegers(asNamespace string, n int) error {
	for i := 0; i < n; i++ {
		key, err := as.NewKey(asNamespace, "integers", i)
		if err != nil {
			return err
		}
		data := as.NewBin("idx", i)
		if err := ac.client.PutBins(nil, key, data); err != nil {
			return err
		}
	}
	return nil
}

func (ac *AerospikeClient) ReadSequentialIntegers(asNamespace string, n int) error {
	for i := 0; i < n; i++ {
		key, err := as.NewKey(asNamespace, "integers", i)
		if err != nil {
			return err
		}
		data, err := ac.client.Get(nil, key)
		if err != nil {
			return err
		}
		if data != nil {
			if b, ok := data.Bins["idx"]; ok {
				if i == b.(int) {
					continue
				}
			}
		}
		return fmt.Errorf("error retrieving idx %d from namespace %s", i, asNamespace)
	}
	return nil
}
