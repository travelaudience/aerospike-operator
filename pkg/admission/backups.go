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

package admission

import (
	"fmt"
	"reflect"

	av1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/backuprestore"
)

func (s *ValidatingAdmissionWebhook) admitAerospikeNamespaceBackup(ar av1beta1.AdmissionReview) *av1beta1.AdmissionResponse {
	// decode the new AerospikeNamespaceBackup object
	obj, err := decodeAerospikeNamespaceBackup(ar.Request.Object.Raw)
	if err != nil {
		return admissionResponseFromError(err)
	}

	// if this is an update to .spec, return an error
	if ar.Request.Operation == av1beta1.Update {
		old, err := decodeAerospikeNamespaceBackup(ar.Request.OldObject.Raw)
		if err != nil {
			return admissionResponseFromError(err)
		}
		// reject the update if the .status field was deleted
		emptyStatus := v1alpha1.AerospikeNamespaceBackupStatus{}
		if !reflect.DeepEqual(old.Status, emptyStatus) && reflect.DeepEqual(obj.Status, emptyStatus) {
			return admissionResponseFromError(fmt.Errorf("the .status field cannot be deleted"))
		}
		// reject updates to the .spec field
		if !reflect.DeepEqual(obj.Spec, old.Spec) {
			return admissionResponseFromError(fmt.Errorf("the spec of an aerospikenamespacebackup resource cannot be changed after creation"))
		}
	}

	// validate the new AerospikeNamespaceBackup
	if err = s.validateBackupRestoreObj(obj); err != nil {
		return admissionResponseFromError(err)
	}

	// admit the AerospikeNamespaceBackup object
	return &av1beta1.AdmissionResponse{Allowed: true}
}

func (s *ValidatingAdmissionWebhook) admitAerospikeNamespaceRestore(ar av1beta1.AdmissionReview) *av1beta1.AdmissionResponse {
	// decode the new AerospikeNamespaceRestore object
	obj, err := decodeAerospikeNamespaceRestore(ar.Request.Object.Raw)
	if err != nil {
		return admissionResponseFromError(err)
	}

	// if this is an update to .spec, return an error
	if ar.Request.Operation == av1beta1.Update {
		old, err := decodeAerospikeNamespaceRestore(ar.Request.OldObject.Raw)
		if err != nil {
			return admissionResponseFromError(err)
		}
		// reject the update if the .status field was deleted
		emptyStatus := v1alpha1.AerospikeNamespaceRestoreStatus{}
		if !reflect.DeepEqual(old.Status, emptyStatus) && reflect.DeepEqual(obj.Status, emptyStatus) {
			return admissionResponseFromError(fmt.Errorf("the .status field cannot be deleted"))
		}
		// reject updates to the .spec field
		if !reflect.DeepEqual(obj.Spec, old.Spec) {
			return admissionResponseFromError(fmt.Errorf("the spec of an aerospikenamespacerestore resource cannot be changed after creation"))
		}
	}

	// validate the new AerospikeNamespaceRestore
	if err = s.validateBackupRestoreObj(obj); err != nil {
		return admissionResponseFromError(err)
	}

	// admit the AerospikeNamespaceBackup object
	return &av1beta1.AdmissionResponse{Allowed: true}
}

func (s *ValidatingAdmissionWebhook) validateBackupRestoreObj(obj v1alpha1.BackupRestoreObject) error {
	// make sure that the target cluster exists
	aerospikeCluster, err := s.aerospikeClient.AerospikeV1alpha1().AerospikeClusters(obj.GetNamespace()).Get(obj.GetTarget().Cluster, v1.GetOptions{})
	if err != nil {
		return err
	}

	// make sure that the target namespace exists
	if !namespaceExists(aerospikeCluster, obj) {
		return fmt.Errorf("cluster %s does not contain a namespace named %s", aerospikeCluster.Name, obj.GetTarget().Namespace)
	}

	// check if object contains BackupStorageSpec and use it. if not
	// try to get it from the cluster. If the later does not contain
	// it, return an error
	var storageSpec *v1alpha1.BackupStorageSpec
	switch {
	case obj.GetStorage() != nil:
		storageSpec = obj.GetStorage()
	case aerospikeCluster.Spec.BackupSpec != nil:
		storageSpec = &aerospikeCluster.Spec.BackupSpec.Storage
	default:
		return fmt.Errorf("must specify .spec.storage")
	}

	// make sure that the secret containing cloud storage credentials exists and
	// matches the expected format
	secret, err := s.kubeClient.CoreV1().Secrets(obj.GetNamespace()).Get(storageSpec.Secret, v1.GetOptions{})
	if err != nil {
		return err
	}
	if _, ok := secret.Data[backuprestore.SecretFilename]; !ok {
		return fmt.Errorf("secret does not contain expected field %q", backuprestore.SecretFilename)
	}

	return nil
}

func namespaceExists(aerospikeCluster *v1alpha1.AerospikeCluster, obj v1alpha1.BackupRestoreObject) bool {
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		if ns.Name == obj.GetTarget().Namespace {
			return true
		}
	}
	return false
}

func decodeAerospikeNamespaceBackup(raw []byte) (*v1alpha1.AerospikeNamespaceBackup, error) {
	obj := &v1alpha1.AerospikeNamespaceBackup{}
	if len(raw) == 0 {
		return obj, nil
	}
	_, _, err := codecs.UniversalDeserializer().Decode(raw, nil, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func decodeAerospikeNamespaceRestore(raw []byte) (*v1alpha1.AerospikeNamespaceRestore, error) {
	obj := &v1alpha1.AerospikeNamespaceRestore{}
	if len(raw) == 0 {
		return obj, nil
	}
	_, _, err := codecs.UniversalDeserializer().Decode(raw, nil, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
