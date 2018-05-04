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

package reconciler

import (
	"bytes"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
	asstrings "github.com/travelaudience/aerospike-operator/pkg/utils/strings"
)

func (r *AerospikeClusterReconciler) ensureConfigMap(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) (*v1.ConfigMap, error) {
	aerospikeConfig := buildConfig(aerospikeCluster)

	desiredConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: aerospikeCluster.Name,
			Labels: map[string]string{
				selectors.LabelAppKey:     selectors.LabelAppVal,
				selectors.LabelClusterKey: aerospikeCluster.Name,
			},
			Namespace: aerospikeCluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         aerospikev1alpha1.SchemeGroupVersion.String(),
					Kind:               kind,
					Name:               aerospikeCluster.Name,
					UID:                aerospikeCluster.UID,
					Controller:         pointers.NewBool(true),
					BlockOwnerDeletion: pointers.NewBool(true),
				},
			},
			Annotations: map[string]string{
				configMapHashLabel: asstrings.Hash(aerospikeConfig),
			},
		},
		Data: map[string]string{configFileName: aerospikeConfig},
	}

	createdConfigMap, err := r.kubeclientset.CoreV1().ConfigMaps(aerospikeCluster.Namespace).Create(desiredConfigMap)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, err
		}

		currentConfigMap, err := r.configMapsLister.ConfigMaps(aerospikeCluster.Namespace).Get(desiredConfigMap.Name)
		if err != nil {
			return nil, err
		}

		mustResetConfig := asstrings.Hash(currentConfigMap.Data[configFileName]) != currentConfigMap.Annotations[configMapHashLabel]
		mustRestartPods := desiredConfigMap.Annotations[configMapHashLabel] != currentConfigMap.Annotations[configMapHashLabel]

		if mustResetConfig || mustRestartPods {
			log.WithFields(log.Fields{
				logfields.AerospikeCluster: meta.Key(aerospikeCluster),
				logfields.ConfigMap:        desiredConfigMap.Name,
			}).Debug("configmap exists but is outdated")

			updatedConfigMap, err := r.kubeclientset.CoreV1().ConfigMaps(aerospikeCluster.Namespace).Update(desiredConfigMap)
			if err != nil {
				return nil, err
			}
			log.WithFields(log.Fields{
				logfields.AerospikeCluster: meta.Key(aerospikeCluster),
				logfields.ConfigMap:        desiredConfigMap.Name,
			}).Debug("configmap updated")
			return updatedConfigMap, nil
		} else {
			log.WithFields(log.Fields{
				logfields.AerospikeCluster: meta.Key(aerospikeCluster),
				logfields.ConfigMap:        desiredConfigMap.Name,
			}).Debug("configmap exists and is up to date")
			return currentConfigMap, nil
		}
	}
	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
		logfields.ConfigMap:        desiredConfigMap.Name,
	}).Debug("configmap created")

	return createdConfigMap, nil
}

func (r *AerospikeClusterReconciler) getConfigMap(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) (*v1.ConfigMap, error) {
	res, err := r.configMapsLister.ConfigMaps(aerospikeCluster.Namespace).Get(aerospikeCluster.Name)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func buildConfig(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) string {
	var namespacesConfig []string

	for _, namespace := range aerospikeCluster.Spec.Namespaces {
		buf := new(bytes.Buffer)
		asNamespaceTemplate.Execute(buf, getNamespaceProps(aerospikeCluster, &namespace))
		namespacesConfig = append(namespacesConfig, buf.String())
	}

	configMapBuffer := new(bytes.Buffer)
	asConfigTemplate.Execute(configMapBuffer, getClusterProps(aerospikeCluster, namespacesConfig))

	return configMapBuffer.String()
}

func getClusterProps(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, namespacesConfig []string) map[string]interface{} {
	return map[string]interface{}{
		clusterMeshServiceKey: aerospikeCluster.Name,
		clusterMeshPortKey:    heartbeatPort,
		clusterNamespacesKey:  namespacesConfig,
	}
}

func getNamespaceProps(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, namespace *aerospikev1alpha1.AerospikeNamespaceSpec) map[string]interface{} {
	props := make(map[string]interface{})

	props[nsNameKey] = namespace.Name

	if namespace.ReplicationFactor <= aerospikeCluster.Spec.NodeCount && namespace.ReplicationFactor > 0 {
		props[nsReplicationFactorKey] = namespace.ReplicationFactor
	} else if namespace.ReplicationFactor > aerospikeCluster.Spec.NodeCount {
		props[nsReplicationFactorKey] = aerospikeCluster.Spec.NodeCount
	}

	if namespace.MemorySize != "" {
		props[nsMemorySizeKey] = namespace.MemorySize
	}

	if value, err := strconv.Atoi(strings.TrimSuffix(namespace.DefaultTTL, "s")); err == nil {
		props[nsDefaultTTLKey] = value
	}

	props[nsStorageTypeKey] = namespace.Storage.Type

	if namespace.Storage.Type == aerospikev1alpha1.StorageTypeFile {
		props[nsStorageSizeKey] = namespace.Storage.Size
		props[nsFilePath] = defaultFilePath
	} else if namespace.Storage.Type == aerospikev1alpha1.StorageTypeDevice {
		props[nsDevicePath] = defaultDevicePath
	}

	return props
}
