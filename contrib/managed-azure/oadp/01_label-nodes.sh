#!/usr/bin/env bash

set -euox pipefail

kubectl label node --all node-role.kubernetes.io/master=""
kubectl label node --all node-role.kubernetes.io/worker=""

