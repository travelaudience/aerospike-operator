#!/bin/sh

# run /aerospike-operator-e2e with the requested parameters
/aerospike-operator-e2e \
	-ginkgo.flakeAttempts="${FLAKE_ATTEMPTS}" \
	-ginkgo.focus="${FOCUS}" \
    -ginkgo.progress \
    -ginkgo.v \
	-gcs-bucket-name="${GCS_BUCKET_NAME}" \
	-gcs-secret-name="${GCS_SECRET_NAME}" \
	-test.timeout="${TIMEOUT}" \
    -test.v
