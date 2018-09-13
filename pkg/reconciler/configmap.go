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

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/common"
	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
	asstrings "github.com/travelaudience/aerospike-operator/pkg/utils/strings"
)

func (r *AerospikeClusterReconciler) ensureConfigMap(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) (*v1.ConfigMap, error) {
	// grab the desired configmap object
	desiredConfigMap := buildConfigMap(aerospikeCluster)
	// try to actually create the configmap resource
	if createdConfigMap, err := r.kubeclientset.CoreV1().ConfigMaps(aerospikeCluster.Namespace).Create(desiredConfigMap); err != nil {
		if errors.IsAlreadyExists(err) {
			// a configmap with the same name already exists, so we need to
			// handle an update
			return r.updateConfigMap(aerospikeCluster, desiredConfigMap)
		}
		return nil, err
	} else {
		// we've got no errors, so we're good to go
		log.WithFields(log.Fields{
			logfields.AerospikeCluster: meta.Key(aerospikeCluster),
			logfields.ConfigMap:        desiredConfigMap.Name,
		}).Debug("configmap created")
		return createdConfigMap, nil
	}
}

func (r *AerospikeClusterReconciler) updateConfigMap(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, desiredConfigMap *v1.ConfigMap) (*v1.ConfigMap, error) {
	// get the current configmap resource
	currentConfigMap, err := r.configMapsLister.ConfigMaps(aerospikeCluster.Namespace).Get(desiredConfigMap.Name)
	if err != nil {
		return nil, err
	}
	// check whether the current configmap resource needs to be updated
	outdated := asstrings.Hash(currentConfigMap.Data[configFileName]) != currentConfigMap.Annotations[configMapHashAnnotation] ||
		desiredConfigMap.Annotations[configMapHashAnnotation] != currentConfigMap.Annotations[configMapHashAnnotation]
	// if the configmap is up-to-date, we're good to go
	if !outdated {
		log.WithFields(log.Fields{
			logfields.AerospikeCluster: meta.Key(aerospikeCluster),
			logfields.ConfigMap:        desiredConfigMap.Name,
		}).Debug("configmap exists and is up to date")
		return currentConfigMap, nil
	}
	// signal that the configmap exists but is outdated
	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
		logfields.ConfigMap:        desiredConfigMap.Name,
	}).Debug("configmap exists but is outdated")
	// update the existing configmap resource to match the desired state
	if updatedConfigMap, err := r.kubeclientset.CoreV1().ConfigMaps(aerospikeCluster.Namespace).Update(desiredConfigMap); err != nil {
		return nil, err
	} else {
		log.WithFields(log.Fields{
			logfields.AerospikeCluster: meta.Key(aerospikeCluster),
			logfields.ConfigMap:        desiredConfigMap.Name,
		}).Debug("configmap updated")
		return updatedConfigMap, nil
	}
}

func buildConfig(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) string {
	var namespacesConfig []string

	for index, namespace := range aerospikeCluster.Spec.Namespaces {
		buf := new(bytes.Buffer)
		asNamespaceTemplate.Execute(buf, getNamespaceProps(aerospikeCluster, index, &namespace))
		namespacesConfig = append(namespacesConfig, buf.String())
	}

	configMapBuffer := new(bytes.Buffer)
	asConfigTemplate.Execute(configMapBuffer, getClusterProps(aerospikeCluster, namespacesConfig))

	return configMapBuffer.String()
}

func buildConfigMap(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) *v1.ConfigMap {
	// build the aerospike config file based on the current spec
	aerospikeConfig := buildConfig(aerospikeCluster)
	// return a configmap object containing aerospikeConfig
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: aerospikeCluster.Name,
			Labels: map[string]string{
				selectors.LabelAppKey:     selectors.LabelAppVal,
				selectors.LabelClusterKey: aerospikeCluster.Name,
			},
			Namespace: aerospikeCluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         aerospikev1alpha2.SchemeGroupVersion.String(),
					Kind:               crd.AerospikeClusterKind,
					Name:               aerospikeCluster.Name,
					UID:                aerospikeCluster.UID,
					Controller:         pointers.NewBool(true),
					BlockOwnerDeletion: pointers.NewBool(true),
				},
			},
			Annotations: map[string]string{
				configMapHashAnnotation: asstrings.Hash(aerospikeConfig),
			},
		},
		Data: map[string]string{configFileName: aerospikeConfig},
	}
}

func getClusterProps(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, namespacesConfig []string) map[string]interface{} {
	return map[string]interface{}{
		serviceNodeIdKey:            ServiceNodeIdValue,
		clusterNamespacesKey:        namespacesConfig,
		heartbeatAddressesConfigKey: HeartbeatAddressesValue,
	}
}

func getNamespaceProps(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, index int, namespace *aerospikev1alpha2.AerospikeNamespaceSpec) map[string]interface{} {
	props := make(map[string]interface{})

	props[nsNameKey] = namespace.Name

	if namespace.ReplicationFactor != nil {
		if *namespace.ReplicationFactor <= aerospikeCluster.Spec.NodeCount && *namespace.ReplicationFactor > 0 {
			props[nsReplicationFactorKey] = namespace.ReplicationFactor
		} else if *namespace.ReplicationFactor > aerospikeCluster.Spec.NodeCount {
			props[nsReplicationFactorKey] = aerospikeCluster.Spec.NodeCount
		}
	}

	if namespace.MemorySize != nil {
		if *namespace.MemorySize != "" {
			props[nsMemorySizeKey] = namespace.MemorySize
		}
	}

	if namespace.DefaultTTL != nil {
		if value, err := strconv.Atoi(strings.TrimSuffix(*namespace.DefaultTTL, "s")); err == nil {
			props[nsDefaultTTLKey] = value
		}
	}

	props[nsStorageTypeKey] = namespace.Storage.Type

	if namespace.Storage.Type == common.StorageTypeFile {
		props[nsStorageSizeKey] = namespace.Storage.Size
		props[nsFilePath] = defaultFilePath
	} else if namespace.Storage.Type == common.StorageTypeDevice {
		props[nsDevicePath] = getIndexBasedDevicePath(index)
	}

	if namespace.Storage.DataInMemory != nil {
		props[nsDataInMemory] = *namespace.Storage.DataInMemory
	}

	return props
}
