#!/bin/sh
# Idempotent MinIO bucket bootstrap. Runs as a one-shot service in the
# Compose graph after MinIO is healthy.
#
# Reads MINIO_* and BUCKET vars from the environment.
set -e

mc alias set local "${MINIO_ENDPOINT_URL:-http://minio:9000}" "${MINIO_ACCESS_KEY:-minioadmin}" "${MINIO_SECRET_KEY:-minioadmin}"
mc mb --ignore-existing "local/${MINIO_BUCKET:-tempest-jobs}"
echo "minio bootstrap: bucket ${MINIO_BUCKET:-tempest-jobs} ready"
