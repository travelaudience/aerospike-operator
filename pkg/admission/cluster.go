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
	"github.com/travelaudience/aerospike-operator/pkg/versioning"
)

const (
	// AerospikeMeshSeedAddressMaxLength represents the maximum length of a name that can be used in
	// the "mesh-seed-address-port", as defined in
	//
	// https://github.com/aerospike/aerospike-server/blob/4.2.0.10/as/src/fabric/hb.c#L6734-L6741
	//
	// In practice, the abovementioned snippet means that the length of
	//
	//     <pod name> + "." + <aerospike cluster name> + "." + <kubernetes namespace name>
	//
	// cannot exceed AerospikeMeshSeedAddressMaxLength (64) characters. Since <pod-name> is created from
	// <aerospike cluster name> by append "-X" (where X is an integer between 0 and 7), this means
	// that 2 * (len(<aerospike cluster name) + 2) + <kubernetes namespace name> cannot exceed this value.
	AerospikeMeshSeedAddressMaxLength = 63
	// AerospikeClusterNameMaxLength represents the maximum length of an AerospikeCluster's metadata.name.
	// the length corresponds to the maximum length of a pod name (63 characters) minus the dash and
	// the index (a single digit).
	AerospikeClusterNameMaxLength = 61
	// aerospikeNamespaceMaxNameLen represents the maximum length of an AerospikeCluster's namespace name.
	// The length corresponds to the maximum length of a pod name (63 characters) minus 40 chars
	// corresponding to the following:
	// -XX.YY.ZZ.WW-XX.YY.ZZ.WW-upgrade-restore, where XX.YY.ZZ.WW appears twice and corresponds to the
	// source and target versions of Aerospike, upgrade is the suffix appended by reconciler and
	// backup/restore suffix is appended to the jobs by backups handler. (restore is used for calculation
	// because it has a greater length)
	aerospikeNamespaceMaxNameLen = 23
	// the default replication factor for an aerospike namespace
	// https://www.aerospike.com/docs/reference/configuration#replication-factor
	defaultNamespaceReplicationFactor int32 = 2
)

func (s *ValidatingAdmissionWebhook) admitAerospikeCluster(ar av1beta1.AdmissionReview) *av1beta1.AdmissionResponse {
	// decode the new AerospikeCluster object
	new, err := decodeAerospikeCluster(ar.Request.Object.Raw)
	if err != nil {
		return admissionResponseFromError(err)
	}
	// decode the old AerospikeCluster object (if any)
	old, err := decodeAerospikeCluster(ar.Request.OldObject.Raw)
	if err != nil {
		return admissionResponseFromError(err)
	}
	// validate the new AerospikeCluster
	if err = s.validateAerospikeCluster(new); err != nil {
		return admissionResponseFromError(err)
	}
	// if this is an update, validate that the transition from old to new
	if ar.Request.Operation == av1beta1.Update {
		if err = s.validateAerospikeClusterUpdate(old, new); err != nil {
			return admissionResponseFromError(err)
		}
	}
	// admit the AerospikeCluster object
	return &av1beta1.AdmissionResponse{Allowed: true}
}

func (s *ValidatingAdmissionWebhook) validateAerospikeCluster(aerospikeCluster *v1alpha1.AerospikeCluster) error {
	// validate that the name doesn't exceed AerospikeClusterNameMaxLength
	if len(aerospikeCluster.Name) > AerospikeClusterNameMaxLength {
		return fmt.Errorf("the name of the cluster cannot exceed %d characters", AerospikeClusterNameMaxLength)
	}

	// validate that we can use pod dns names as "mesh-seed-address-port" entries
	if 2*(len(aerospikeCluster.Name)+2)+len(aerospikeCluster.Namespace) > AerospikeMeshSeedAddressMaxLength {
		return fmt.Errorf("the current combination of cluster and kubernetes namespace names cannot be used")
	}

	// validate the Aerospike version
	if version, err := versioning.NewVersionFromString(aerospikeCluster.Spec.Version); err != nil {
		return err
	} else if !version.IsSupported() {
		return fmt.Errorf("aerospike version %q is not supported", aerospikeCluster.Spec.Version)
	}

	// enforce the existence of a single namespace per cluster
	if len(aerospikeCluster.Spec.Namespaces) != 1 {
		return fmt.Errorf("the number of namespaces in the cluster must be exactly one")
	}

	// validate every namespace's name and that its replication factor
	// is less than or equal to the cluster's node count.
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		if len(ns.Name) > aerospikeNamespaceMaxNameLen {
			return fmt.Errorf("the name of a namespace cannot exceed %d characters", aerospikeNamespaceMaxNameLen)
		}
		// the current replication factor equals aerospike's default, unless it
		// has been set by the user
		currentReplicationFactor := defaultNamespaceReplicationFactor
		if ns.ReplicationFactor != nil {
			currentReplicationFactor = *ns.ReplicationFactor
		}
		if currentReplicationFactor > aerospikeCluster.Spec.NodeCount {
			return fmt.Errorf("replication factor of %d requested for namespace %s but the cluster has only %d nodes", currentReplicationFactor, ns.Name, aerospikeCluster.Spec.NodeCount)
		}
	}

	// if backupSpec is specified, make sure that the secret containing
	// cloud storage credentials exists and matches the expected format
	if aerospikeCluster.Spec.BackupSpec != nil {
		secret, err := s.kubeClient.CoreV1().Secrets(aerospikeCluster.Namespace).Get(aerospikeCluster.Spec.BackupSpec.Storage.Secret, v1.GetOptions{})
		if err != nil {
			return err
		}
		if _, ok := secret.Data[backuprestore.SecretFilename]; !ok {
			return fmt.Errorf("secret does not contain expected field %q", backuprestore.SecretFilename)
		}
	}
	return nil
}

func (s *ValidatingAdmissionWebhook) validateAerospikeClusterUpdate(old, new *v1alpha1.AerospikeCluster) error {
	// reject the update if the .status field was deleted
	emptyStatus := v1alpha1.AerospikeClusterStatus{}
	if !reflect.DeepEqual(old.Status, emptyStatus) && reflect.DeepEqual(new.Status, emptyStatus) {
		return fmt.Errorf("the .status field cannot be deleted")
	}

	// check whether a version upgrade has been requested, in which case we
	// prevent configuration/topology changes from occurring simultaneously
	if old.Spec.Version != new.Spec.Version {
		// create a copy of the new spec
		tmp := new.DeepCopy()
		// set tmp.Spec.Version to old.Spec.Version
		tmp.Spec.Version = old.Spec.Version
		// check if old.Spec and tmp.Spec differ
		// if they do, more than just .spec.Version has been been changed
		// between old and new, and new must be rejected
		if !reflect.DeepEqual(old.Spec, tmp.Spec) {
			return fmt.Errorf("when changing .spec.version no other changes to .spec can be performed")
		}
		// fail if the aerospikecluster resource doesn't contain .spec.backupSpec
		if new.Spec.BackupSpec == nil {
			return fmt.Errorf("no value for .spec.backupSpec has been specified")
		}
	}

	// validate the transition between old.spec.version and new.spec.version
	if err := validateVersion(old, new); err != nil {
		return err
	}
	// validate the namespace configuration
	if err := validateNamespaces(old, new); err != nil {
		return err
	}

	return nil
}

func validateVersion(old, new *v1alpha1.AerospikeCluster) error {
	// if the version was not changed, we're good
	if old.Spec.Version == new.Spec.Version {
		return nil
	}
	// validate the requested version transition
	sourceVersion, err := versioning.NewVersionFromString(old.Spec.Version)
	if err != nil {
		return err
	}
	targetVersion, err := versioning.NewVersionFromString(new.Spec.Version)
	if err != nil {
		return err
	}
	upgrade := versioning.VersionUpgrade{sourceVersion, targetVersion}
	// return an error if the transition is not supported
	if !upgrade.IsValid() {
		return fmt.Errorf("cannot upgrade from version %v to %v", sourceVersion, targetVersion)
	}
	return nil
}

func validateNamespaces(old, new *v1alpha1.AerospikeCluster) error {
	// grab a name => spec map for the namespaces in the old object
	oldnss := namespaceMap(old)
	// grab a name => spec map for the namespaces in the new object
	newnss := namespaceMap(new)
	// prevent two namespaces with the same name from appearing in the spec
	if len(newnss) < len(new.Spec.Namespaces) {
		return fmt.Errorf("namespace names must be unique")
	}
	// validate that no namespace has been removed
	for name := range oldnss {
		if _, ok := newnss[name]; !ok {
			return fmt.Errorf("cannot remove namespace %s", name)
		}
	}
	// validate that there were no changes to existing namespaces
	for name := range newnss {
		// if the namespace didn't exist before, there's nothing to validate
		if _, ok := oldnss[name]; !ok {
			continue
		}
		// make sure that the replication factor hasn't been changed
		if oldnss[name].ReplicationFactor != nil && newnss[name].ReplicationFactor != nil && *oldnss[name].ReplicationFactor != *newnss[name].ReplicationFactor {
			return fmt.Errorf("cannot change the replication factor for namespace %s", name)
		}
		// make sure that the storage spec hasn't been changed
		if !reflect.DeepEqual(oldnss[name].Storage, newnss[name].Storage) {
			return fmt.Errorf("cannot change the storage spec for namespace %s", name)
		}
	}
	return nil
}

func namespaceMap(aerospikeCluster *v1alpha1.AerospikeCluster) map[string]v1alpha1.AerospikeNamespaceSpec {
	res := make(map[string]v1alpha1.AerospikeNamespaceSpec, len(aerospikeCluster.Spec.Namespaces))
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		res[ns.Name] = ns
	}
	return res
}

func decodeAerospikeCluster(raw []byte) (*v1alpha1.AerospikeCluster, error) {
	obj := &v1alpha1.AerospikeCluster{}
	if len(raw) == 0 {
		return obj, nil
	}
	_, _, err := codecs.UniversalDeserializer().Decode(raw, nil, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
