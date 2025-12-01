#!/usr/bin/env bash

set -euox pipefail

until kubectl get serviceaccount openshift-adp-controller-manager -n openshift-adp &>/dev/null; do
    echo "Waiting for serviceaccount openshift-adp-controller-manager to exist..."
    sleep 5
done

kubectl patch serviceaccount -n openshift-adp openshift-adp-controller-manager -p '{"imagePullSecrets": [{"name": "pull-secret"}]}'

oc delete pod -n openshift-adp --all
