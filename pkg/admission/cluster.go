package admission

import (
	"fmt"
	"reflect"

	av1beta1 "k8s.io/api/admission/v1beta1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/backuprestore"
	"github.com/travelaudience/aerospike-operator/pkg/versioning"
)

const (
	// aerospikeClusterMaxNameLen represents the maximum length of an AerospikeCluster's metadata.name.
	aerospikeClusterMaxNameLen = 61
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
	// validate that the name doesn't exceed 61 characters
	if len(aerospikeCluster.Name) > aerospikeClusterMaxNameLen {
		return fmt.Errorf("the name of the cluster cannot exceed %d characters", aerospikeClusterMaxNameLen)
	}

	// validate that every namespace's replication factor is less than or equal to the cluster's node count.
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		if ns.ReplicationFactor > aerospikeCluster.Spec.NodeCount {
			return fmt.Errorf("replication factor of %d requested for namespace %s but the cluster has only %d nodes", ns.ReplicationFactor, ns.Name, aerospikeCluster.Spec.NodeCount)
		}
	}

	// if backupSpec is specified, make sure that the secret containing
	// cloud storage credentials exists and matches the expected format
	if aerospikeCluster.Spec.BackupSpec != nil {
		secret, err := s.secretsLister.Secrets(aerospikeCluster.Namespace).Get(aerospikeCluster.Spec.BackupSpec.Storage.Secret)
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
		if _, ok := oldnss[name]; !ok {
			continue
		}
		if !reflect.DeepEqual(oldnss[name], newnss[name]) {
			return fmt.Errorf("cannot change the spec for namespace %s", name)
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
