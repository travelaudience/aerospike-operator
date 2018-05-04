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
	"time"

	as "github.com/aerospike/aerospike-client-go"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
)

type AerospikeClient struct {
	client *as.Client
}

func NewAerospikeClient(aerospikeCluster *v1alpha1.AerospikeCluster) (*AerospikeClient, error) {
	svc := fmt.Sprintf("%s.%s", aerospikeCluster.Name, aerospikeCluster.Namespace)
	c, err := as.NewClientWithPolicy(as.NewClientPolicy(), svc, 3000)
	if err != nil {
		return nil, err
	}

	// set SocketTimeout to 0
	// https://github.com/aerospike/aerospike-client-go/issues/227
	// https://github.com/aerospike/aerospike-client-go/issues/229
	c.DefaultPolicy.SocketTimeout = 0
	c.DefaultWritePolicy.SocketTimeout = 0

	c.DefaultWritePolicy.Timeout = 3 * time.Second
	c.DefaultWritePolicy.MaxRetries = 2
	c.DefaultWritePolicy.SleepBetweenRetries = 1 * time.Second

	return &AerospikeClient{client: c}, nil
}

func (ac *AerospikeClient) Close() {
	ac.client.Close()
}

func (ac *AerospikeClient) IsConnected() bool {
	return ac.client.IsConnected()
}

func (ac *AerospikeClient) WriteSequentialIntegers(asNamespace string, n int) error {
	for i := 1; i <= n; i++ {
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
	for i := 1; i <= n; i++ {
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
