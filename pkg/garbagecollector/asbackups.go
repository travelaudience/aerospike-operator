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

package garbagecollector

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/common"
	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	aerospikelisters "github.com/travelaudience/aerospike-operator/pkg/client/listers/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	astime "github.com/travelaudience/aerospike-operator/pkg/utils/time"
)

type AerospikeNamespaceBackupHandler struct {
	kubeclientset                  kubernetes.Interface
	aerospikeclientset             aerospikeclientset.Interface
	aerospikeNamespaceBackupLister aerospikelisters.AerospikeNamespaceBackupLister
	recorder                       record.EventRecorder
}

func NewAerospikeNamespaceBackupHandler(kubeclientset kubernetes.Interface,
	aerospikeclientset aerospikeclientset.Interface,
	aerospikeNamespaceBackupLister aerospikelisters.AerospikeNamespaceBackupLister,
	recorder record.EventRecorder) *AerospikeNamespaceBackupHandler {
	return &AerospikeNamespaceBackupHandler{
		kubeclientset:                  kubeclientset,
		aerospikeclientset:             aerospikeclientset,
		aerospikeNamespaceBackupLister: aerospikeNamespaceBackupLister,
		recorder:                       recorder,
	}
}

func (h *AerospikeNamespaceBackupHandler) Handle(asBackup *aerospikev1alpha2.AerospikeNamespaceBackup) error {
	log.WithFields(log.Fields{
		logfields.Key: meta.Key(asBackup),
	}).Debug("checking whether aerospikenamespacebackup has expired")

	// get the corresponding aerospikecluster object
	aerospikeCluster, err := h.aerospikeclientset.AerospikeV1alpha2().AerospikeClusters(asBackup.Namespace).Get(asBackup.Spec.Target.Cluster, v1.GetOptions{})
	if err != nil {
		return err
	}

	// skip aerospikenamespacebackup if no TTL was set
	if asBackup.Spec.TTL == nil {
		if aerospikeCluster.Spec.BackupSpec != nil {
			asBackup.Spec.TTL = aerospikeCluster.Spec.BackupSpec.TTL
		}
		if asBackup.Spec.TTL == nil {
			return nil
		}
	}

	// get the aerospikenamespacebackup object expiration as a
	// duration object
	objExpiration, err := astime.ParseDuration(*asBackup.Spec.TTL)
	if err != nil {
		return err
	}

	// check if the aerospikenamespacebackup object expiration
	// has no duration, in which case we return immediately
	if objExpiration == time.Second*0 {
		log.WithFields(log.Fields{
			logfields.Key: meta.Key(asBackup),
		}).Debug("no expiration set for aerospikenamespacebackup")
		return nil
	}

	// check if aerospikenamespacebackup object has expired
	if time.Now().After(asBackup.CreationTimestamp.Add(objExpiration)) {
		// get backupStorage spec from target aerospikecluster
		// if not available in aerospikenamespacebackup resource.
		if asBackup.Spec.Storage == nil {
			asBackup.Spec.Storage = &aerospikeCluster.Spec.BackupSpec.Storage
			if asBackup.Spec.Storage == nil {
				return fmt.Errorf("backupstorage not specified on aerospikenamespacebackup or aerospikecluster")
			}
		}

		// delete backup data from cloud storage
		switch asBackup.Spec.Storage.Type {
		case common.StorageTypeGCS:
			if err := h.deleteBackupDataGCS(asBackup); err != nil {
				log.WithFields(log.Fields{
					logfields.Key: meta.Key(asBackup),
				}).Infof("could not delete backup data from cloud storage: %s", err)
			}
			log.WithFields(log.Fields{
				logfields.Key: meta.Key(asBackup),
			}).Info("backup data deleted from cloud storage")
		default:
			return fmt.Errorf("storage type not supported")
		}

		// delete AerospikeNamespaceBackup resource
		if err := h.aerospikeclientset.AerospikeV1alpha2().AerospikeNamespaceBackups(asBackup.Namespace).Delete(asBackup.Name, &v1.DeleteOptions{}); err != nil {
			return err
		}
		log.WithFields(log.Fields{
			logfields.Key: meta.Key(asBackup),
		}).Info("expired aerospikenamespacebackup deleted by garbage collector")
	}

	return nil
}
