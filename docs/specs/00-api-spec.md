# API Types

This document describes the types introduced by aerospike-operator.

## Table of Contents

* [Base Types](#base-types)
  * [AerospikeCluster](#aerospikecluster)
  * [AerospikeNamespace](#aerospikenamespace)
  * [AerospikeNamespaceBackup](#aerospikenamespacebackup)
  * [AerospikeNamespaceRestore](#aerospikenamespacerestore)
* [Nested Types](#nested-types)
  * [AerospikeClusterSpec](#aerospikeclusterspec)
  * [AerospikeNamespaceSpec](#aerospikenamespacespec)
  * [StorageSpec](#storagespec)
  * [FileStorageSpec](#filestoragespec)
  * [DeviceStorageSpec](#devicestoragespec)
  * [AerospikeNamespaceBackupSpec](#aerospikenamespacebackupspec)
  * [AerospikeNamespaceRestoreSpec](#aerospikenamespacerestorespec)
  * [TargetCluster](#targetcluster)
  * [GCSStorageSpec](#gcsstoragespec)
* [Status Types](#status-types)

# Base Types

## AerospikeCluster

AerospikeCluster defines an Aerospike Cluster.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata | Standard object metadata. | [metav1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.9/#objectmeta-v1-meta) | true |
| spec | Specification of the desired state of the Aerospike cluster. | [AerospikeClusterSpec](#aerospikeclusterspec) | true |

More info:
* https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
* https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status

### Example

```yaml
apiVersion: aerospike/v1alpha1
kind: AerospikeCluster
metadata:
    name: example-aerospike-cluster
    namespace: example-namespace
spec:
    version: "4.0.0.4"
    replicas: 3
```

[Back to TOC](#table-of-contents)

## AerospikeNamespace

AerospikeNamespace defines an Aerospike namespace.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata | Standard object metadata. | [metav1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.9/#objectmeta-v1-meta) | true |
| spec | Specification of the desired configuration of the Aerospike namespace. | [AerospikeNamespaceSpec](#aerospikenamespacespec) | true |

More info:
* https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
* https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status
* https://www.aerospike.com/docs/operations/configure/namespace

### Example

```yaml
apiVersion: aerospike/v1alpha1
kind: AerospikeNamespace
metadata:
    name: example-aerospike-namespace
    namespace: example-namespace
spec:
    cluster: example-aerospike-cluster
    replicationFactor: 2
    memorySize: 4G
    defaultTTL: 0
    storage:
        type: file
        file:
            files:
            - "/path/to/file1"
            - "/path/to/file2"
```

[Back to TOC](#table-of-contents)

## AerospikeNamespaceBackup

Specification of a single backup operation of an Aerospike namespace.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata | Standard object metadata. | [metav1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.9/#objectmeta-v1-meta) | true |
| spec | Specification of the desired configuration for the backup. | [AerospikeNamespaceBackupSpec](#aerospikenamespacebackupspec) | true |

More info:
* https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
* https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status
* https://www.aerospike.com/docs/tools/backup

### Example

```yaml
apiVersion: aerospike/v1alpha1
kind: AerospikeNamespaceBackup
metadata:
    name: example-aerospike-backup
    namespace: example-namespace
spec:
    target:
        cluster: example-aerospike-cluster
        namespace: example-aerospike-namespace
    storageType: gcs
    gcs:
        bucket: bucket-name
        secret: secret-name
```

[Back to TOC](#table-of-contents)

## AerospikeNamespaceRestore

Specification of a single restore operation of an Aerospike namespace.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata | Standard object metadata. | [metav1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.9/#objectmeta-v1-meta) | true |
| spec | Specification of the desired configurations for restoring from backup. | [AerospikeNamespaceRestoreSpec](#aerospikenamespacerestorespec) | true |

More info:
* https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
* https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status
* https://www.aerospike.com/docs/tools/backup

### Example

```yaml
apiVersion: aerospike/v1alpha1
kind: AerospikeNamespaceRestore
metadata:
    name: example-aerospike-restore
    namespace: example-namespace
spec:    
    target:
        cluster: example-aerospike-cluster
        namespace: example-aerospike-namespace
    storageType: gcs
    gcs:
        bucket: bucket-name
        secret: secret-credentials
```

[Back to TOC](#table-of-contents)

# Nested Types

## AerospikeClusterSpec

Specification of the desired state of the Aerospike cluster.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| version | Version of Aerospike to be deployed. | string | true |
| replicas | Number of instances of Aerospike that should be deployed. | int32 | true |

[Back to TOC](#table-of-contents)

## AerospikeNamespaceSpec

Specification of the desired configuration for the Aerospike namespace.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| cluster | The name of the cluster in which this namespace will be created. | string | true |
| replicationFactor | Specifies the number of replicas in which the namespace should have copies of the data. | int32 | false |
| memorySize | Amount of memory to be used for index and data. Supports suffixes _K_, _M_, _G_, _T_ and _P_. | string | false |
| defaultTTL | Default time-to-live (in seconds) for a record from the time of creation or last update. | int32 | false |
| storage | Specifies configuration properties for the storage to be used by the namespace. | [StorageSpec](#storagespec) | true |

More info:
* https://www.aerospike.com/docs/reference/configuration

[Back to TOC](#table-of-contents)

## StorageSpec

Specifies the type of storage to be used in the Aerospike namespace.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| type | The storage engine to use for the namespace (`file` or `device`). | string | true |
| file | Specification of file storage. | [FileStorageSpec](#filestoragespec) | false |
| device | Specification of device storage. | [DeviceStorageSpec](#devicestoragespec) | false |

More info:
* https://www.aerospike.com/docs/reference/configuration

[Back to TOC](#table-of-contents)

## FileStorageSpec

Configuration of file storage for an Aerospike namespace.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| files | List of paths to the files that will store this namespace's data. | []string | true |

More info:
* https://www.aerospike.com/docs/reference/configuration

[Back to TOC](#table-of-contents)

## DeviceStorageSpec

Configuration of device storage for an Aerospike namespace.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| devices | List of paths to the devices that will store this namespace's data. | []string | true |

More info:
* https://www.aerospike.com/docs/reference/configuration

[Back to TOC](#table-of-contents)

## AerospikeNamespaceBackupSpec

Specification of the desired configurations for a backup.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| target | Specifies which cluster and namespace the backup will target. | [TargetCluster](#targetcluster) | true |
| storageType | Indicates the type of storage in which the backup will be stored. | string | true |
| gcs | Specifies configuration properties for storing in GCS storage. | [GCSStorageSpec](#gcsstoragespec) | false |

More info:
* https://www.aerospike.com/docs/tools/backup

[Back to TOC](#table-of-contents)

## AerospikeNamespaceRestoreSpec

Specification of the desired configurations for restoring from backup.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| target | Specifies to which cluster and namespace the backup will be restored. | [TargetCluster](#targetcluster) | true |
| storageType | Indicates the type of storage from which the backup will be retrieved. | string | true |
| gcs | Specifies configuration properties for retrieving from GCS storage. | *[GCSStorageSpec](#gcsstoragespec) | false |

More info:
* https://www.aerospike.com/docs/tools/backup

[Back to TOC](#table-of-contents)

## TargetCluster

Specification of the cluster and namespace a single backup or restore operation will target.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| cluster | The name of the cluster in which we are performing the backup/restore. | string | true |
| namespace | The name of the namespace to backup/restore. | string | true |

[Back to TOC](#table-of-contents)

## GCSStorageSpec

Specification of configuration properties for GCS storage.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| bucket | The name of the GCS bucket where a given backup is stored. | string | true |
| secret | The name of the secret containing credentials to access the bucket. This secret must contain a GCP service account private key in JSON format. | string | true |

[Back to TOC](#table-of-contents)

# Status Types

The following base types have an associated _status_ type whose structure
mirrors the type's _spec_:

* AerospikeCluster
* AerospikeNamespace
* AerospikeNamespaceBackup
* AerospikeNamespaceRestore

The _status_ type is used to report information about a resource's most recently
observed status. Some of these _status_ types may include the following extra
field:

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| phase | The current phase of the target resource | string | false |

For instance, the value of the _phase_ field in an AerospikeClusterStatus may be
one of the following:

| Phase | Description |
| ----- | ----------- |
| CREATING | The cluster is being deployed. |
| RUNNING | The cluster is running.  |
| SCALING | The cluster is scaling (either up or down). |
| TERMINATING | The cluster is being terminated. |

[Back to TOC](#table-of-contents)
