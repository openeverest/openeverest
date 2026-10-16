#!/usr/bin/env bash
set -euo pipefail

namespace=core-e2e-test
instance=core-smoke
storage=core-smoke-storage
bucket=bucket-1
endpoint=https://127.0.0.1:18333

export AWS_ACCESS_KEY_ID=seaweedfsadmin
export AWS_SECRET_ACCESS_KEY=seaweedfsadmin
export AWS_DEFAULT_REGION=us-east-1
export AWS_EC2_METADATA_DISABLED=true

wait_for_field() {
  local kind=$1 name=$2 path=$3 expected=$4 current
  for ((attempt=0; attempt<90; attempt++)); do
    current=$(kubectl -n "$namespace" get "$kind" "$name" -o "jsonpath={$path}" 2>/dev/null || true)
    if [[ "$current" == "$expected" ]]; then
      return 0
    fi
    if [[ "$current" == Failed || "$current" == Error ]]; then
      echo "$kind/$name reached $current while waiting for $expected" >&2
      kubectl -n "$namespace" get "$kind" "$name" -o yaml >&2
      return 1
    fi
    sleep 2
  done
  echo "Timed out waiting for $kind/$name $path=$expected; last value: $current" >&2
  kubectl -n "$namespace" get "$kind" "$name" -o yaml >&2
  return 1
}

object_count() {
  aws --endpoint-url "$endpoint" --no-verify-ssl s3api list-objects-v2 \
    --bucket "$bucket" --prefix "$1" --query KeyCount --output text 2>/dev/null
}

expect_object_count() {
  local key=$1 expected=$2 actual
  actual=$(object_count "$key")
  if [[ "$actual" != "$expected" ]]; then
    echo "Expected $expected S3 objects for $key, found $actual" >&2
    return 1
  fi
}

kubectl -n seaweedfs rollout status deployment/seaweedfs --timeout=120s
kubectl -n seaweedfs port-forward svc/seaweedfs 18333:443 >/tmp/core-e2e-s3-forward.log 2>&1 &
forward_pid=$!
trap 'kill "$forward_pid" 2>/dev/null || true' EXIT
for ((attempt=0; attempt<30; attempt++)); do
  if aws --endpoint-url "$endpoint" --no-verify-ssl s3api head-bucket --bucket "$bucket" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
aws --endpoint-url "$endpoint" --no-verify-ssl s3api head-bucket --bucket "$bucket" >/dev/null 2>&1

kubectl create namespace "$namespace"
kubectl -n "$namespace" create secret generic core-smoke-s3 \
  --from-literal=AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" \
  --from-literal=AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY"
kubectl -n "$namespace" apply -f - <<EOF
apiVersion: backup.openeverest.io/v1alpha1
kind: BackupStorage
metadata:
  name: $storage
spec:
  type: s3
  s3:
    bucket: $bucket
    region: us-east-1
    endpointURL: https://seaweedfs.seaweedfs.svc.cluster.local
    forcePathStyle: true
    verifyTLS: false
    credentialsSecretRef:
      name: core-smoke-s3
---
apiVersion: core.openeverest.io/v1alpha1
kind: Instance
metadata:
  name: $instance
spec:
  providerRef:
    name: core-e2e-test
  backup:
    enabled: true
    classRef:
      name: core-e2e-test
    storages:
      - storageRef:
          name: $storage
EOF
wait_for_field instance "$instance" .status.phase Ready

kubectl -n "$namespace" annotate instance "$instance" test.openeverest.io/control=fail
wait_for_field instance "$instance" .status.phase Failed
kubectl -n "$namespace" annotate instance "$instance" test.openeverest.io/control-
for ((attempt=0; attempt<90; attempt++)); do
  [[ $(kubectl -n "$namespace" get instance "$instance" -o jsonpath='{.status.phase}') == Ready ]] && break
  sleep 2
done
wait_for_field instance "$instance" .status.phase Ready

create_backup() {
  local name=$1 policy=$2
  kubectl -n "$namespace" apply -f - <<EOF
apiVersion: backup.openeverest.io/v1alpha1
kind: Backup
metadata:
  name: $name
spec:
  origin:
    type: Instance
    instanceRef:
      name: $instance
  classRef:
    name: core-e2e-test
  storageRef:
    name: $storage
  deletionPolicy: $policy
EOF
  wait_for_field backup "$name" .status.state Succeeded
}

backup_key() {
  local uid
  uid=$(kubectl -n "$namespace" get backup "$1" -o jsonpath='{.metadata.uid}')
  echo "core-e2e-test/$namespace/$instance/$uid"
}

create_backup core-smoke-delete Delete
delete_key=$(backup_key core-smoke-delete)
expect_object_count "$delete_key" 1

kubectl -n "$namespace" apply -f - <<EOF
apiVersion: backup.openeverest.io/v1alpha1
kind: Restore
metadata:
  name: core-smoke-restore
spec:
  instanceRef:
    name: $instance
  dataSource:
    type: Backup
    backup:
      backupRef:
        name: core-smoke-delete
EOF
wait_for_field restore core-smoke-restore .status.state Succeeded
kubectl -n "$namespace" delete restore core-smoke-restore --wait=true --timeout=120s

kubectl -n "$namespace" apply -f - <<EOF
apiVersion: core.openeverest.io/v1alpha1
kind: Instance
metadata:
  name: core-smoke-seeded
spec:
  providerRef:
    name: core-e2e-test
  backup:
    enabled: true
    classRef:
      name: core-e2e-test
    storages:
      - storageRef:
          name: $storage
  dataSource:
    type: Backup
    backup:
      backupRef:
        name: core-smoke-delete
EOF
wait_for_field restore core-smoke-seeded-datasource .status.state Succeeded
wait_for_field instance core-smoke-seeded .status.phase Ready
kubectl -n "$namespace" delete instance core-smoke-seeded --wait=true --timeout=120s
kubectl -n "$namespace" delete backup core-smoke-delete --wait=true --timeout=120s
expect_object_count "$delete_key" 0

kubectl -n "$namespace" apply -f - <<EOF
apiVersion: backup.openeverest.io/v1alpha1
kind: Backup
metadata:
  name: core-smoke-fail
  annotations:
    test.openeverest.io/control: fail
spec:
  origin:
    type: Instance
    instanceRef:
      name: $instance
  classRef:
    name: core-e2e-test
  storageRef:
    name: $storage
EOF
wait_for_field backup core-smoke-fail .status.state Failed
kubectl -n "$namespace" delete backup core-smoke-fail --wait=true --timeout=120s

create_backup core-smoke-retain Retain
retain_key=$(backup_key core-smoke-retain)
expect_object_count "$retain_key" 1
kubectl -n "$namespace" delete backup core-smoke-retain --wait=true --timeout=120s
expect_object_count "$retain_key" 1

create_backup core-smoke-cascade Delete
cascade_key=$(backup_key core-smoke-cascade)
expect_object_count "$cascade_key" 1
kubectl -n "$namespace" delete instance "$instance" --wait=true --timeout=120s
kubectl -n "$namespace" wait --for=delete backup/core-smoke-cascade --timeout=120s
expect_object_count "$cascade_key" 0

echo "Core provider smoke passed: readiness, failure, backup, restore, seeding, Delete, Retain, and cascade."
