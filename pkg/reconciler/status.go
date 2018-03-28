/*
Copyright 2018 The aerospike-controller Authors.

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

package reconciler

import (
	"encoding/json"
	"reflect"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
)

type jsonPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

func (r *AerospikeClusterReconciler) updateStatus(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) error {
	patch := make([]jsonPatch, 0)
	if aerospikeCluster.Status.Version == "" {
		patch = append(patch, jsonPatch{
			Op:   "add",
			Path: "/status",
			Value: map[string]interface{}{
				"version":    aerospikeCluster.Spec.Version,
				"namespaces": aerospikeCluster.Spec.Namespaces,
			},
		})
	} else {
		if aerospikeCluster.Status.Version != aerospikeCluster.Spec.Version {
			patch = append(patch, jsonPatch{
				Op:    "add",
				Path:  "/status/version",
				Value: aerospikeCluster.Spec.Version,
			})
		}
		if !reflect.DeepEqual(aerospikeCluster.Status.Namespaces, aerospikeCluster.Spec.Namespaces) {
			patch = append(patch, jsonPatch{
				Op:    "add",
				Path:  "/status/namespaces",
				Value: aerospikeCluster.Spec.Namespaces,
			})
		}
		pods, err := r.listPodsOwnedBy(aerospikeCluster)
		if err != nil {
			return err
		}
		if aerospikeCluster.Status.NodeCount != len(pods) {
			patch = append(patch, jsonPatch{
				Op:    "add",
				Path:  "/status/nodeCount",
				Value: len(pods),
			})
		}
	}

	if len(patch) == 0 {
		log.WithFields(log.Fields{
			logfields.AerospikeCluster: meta.Key(aerospikeCluster),
		}).Debug("no changes to status")
		return nil
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	_, err = r.aerospikeclientset.AerospikeV1alpha1().AerospikeClusters(aerospikeCluster.Namespace).Patch(aerospikeCluster.Name, types.JSONPatchType, patchBytes)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debug("updated status")

	return nil
}
