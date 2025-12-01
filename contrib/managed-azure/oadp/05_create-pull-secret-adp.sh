#!/usr/bin/env bash

kubectl delete secret pull-secret -n openshift-adp || true
kubectl create secret generic pull-secret \
  --from-file=.dockerconfigjson=$PULL_SECRET \
  --type=kubernetes.io/dockerconfigjson \
  --namespace=openshift-adp