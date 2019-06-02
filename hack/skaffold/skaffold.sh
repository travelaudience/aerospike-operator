#!/bin/bash

set -e

# base64 needs different args depending on the flavor of the tool that is installed.
base64w () {
    (base64 --version >/dev/null 2>&1 && base64 -w 0) || base64 --break 0
}

# sed needs different args to -i depending on the flavor of the tool that is installed.
sedi () {
    (sed --version >/dev/null 2>&1 && sed -i "$@") || sed -i "" "$@"
}

# NAMESPACE is the namespace where to deploy the build artifacts.
NAMESPACE=${NAMESPACE:-aerospike-operator}
# ROOT_DIR is the absolute path to the root of the repository.
ROOT_DIR="$(git rev-parse --show-toplevel)"
# TARGET is one of "e2e" or "operator".
TARGET="${TARGET:-operator}"
# TMP_DIR is the path (relative to ROOT_DIR) where to copy manifest templates to.
TMP_DIR="tmp/skaffold/${TARGET}"

# STORAGE_ADMIN_KEY_JSON_FILE is the path to the file containing the credentials of the IAM service account with "roles/storage.admin" role.
STORAGE_ADMIN_KEY_JSON_FILE="${STORAGE_ADMIN_KEY_JSON_FILE:-${ROOT_DIR}/key.json}"
# PROFILE is the skaffold profile to use.
PROFILE=${PROFILE:-minikube}
# PROJECT_ID is the ID of the Google Cloud Platform project where aerospike-operator should be deployed to.
PROJECT_ID=${PROJECT_ID:-aerospike-operator}

# Switch directories to "ROOT_DIR".
pushd "${ROOT_DIR}" > /dev/null

# Create the temporary directory if it does not exist.
mkdir -p "${TMP_DIR}"
# Copy manifest templates to the temporary directory.
cp -r "${ROOT_DIR}/docs/examples/00-prereqs.yml" "${TMP_DIR}/"
cp -r "${ROOT_DIR}/hack/skaffold/${TARGET}/"* "${TMP_DIR}/"

# Replace the "__TMP_DIR__" placeholder.
sedi -e "s|__TMP_DIR__|${TMP_DIR}|" "${TMP_DIR}/"*.yaml
# Replace the "__PROJECT_ID__" placeholder.
sedi -e "s|__PROJECT_ID__|${PROJECT_ID}|g" "${TMP_DIR}/"*.yaml
# Replace the "__BASE64_ENCODED_ADMIN_KEY_JSON__" placeholder.
BASE64_ENCODED_STORAGE_ADMIN_KEY_JSON="$(base64w < "${STORAGE_ADMIN_KEY_JSON_FILE}")"
sedi -e "s|__BASE64_ENCODED_STORAGE_ADMIN_KEY_JSON__|${BASE64_ENCODED_STORAGE_ADMIN_KEY_JSON}|g" "${TMP_DIR}/"*.yaml
# Replace the "__GCS_BUCKET_NAME__" placeholder.
sedi -e "s|__GCS_BUCKET_NAME__|${GCS_BUCKET_NAME}|g" "${TMP_DIR}/"*.yaml
# Replace the "__FOCUS__" placeholder.
sedi -e "s|__FOCUS__|${FOCUS}|g" "${TMP_DIR}/"*.yaml
# Replace the "__SKIP__" placeholder.
sedi -e "s|__SKIP__|${SKIP}|g" "${TMP_DIR}/"*.yaml

# Build the required binary.
case "${TARGET}" in
    "e2e")
        make -C "${ROOT_DIR}" test.e2e.build
	;;
    "operator")
        make -C "${ROOT_DIR}" build BIN="operator"
	;;
esac

# Make sure the target namespace exists.
kubectl get namespace "${NAMESPACE}" > /dev/null 2>&1 || kubectl create namespace "${NAMESPACE}"

# Run skaffold.
skaffold run --tail -f "${TMP_DIR}/skaffold.yaml" -n "${NAMESPACE}" -p "${PROFILE}"
