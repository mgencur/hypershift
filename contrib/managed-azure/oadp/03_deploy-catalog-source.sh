#!/usr/bin/env bash

set -uox pipefail

export PULL_SECRET=/home/mgencur/hypershift/mgencur-pullsecret/pullsecret.json

kubectl delete secret pull-secret -n olm || true
kubectl create secret generic pull-secret \
  --from-file=.dockerconfigjson=$PULL_SECRET \
  --type=kubernetes.io/dockerconfigjson \
  --namespace=olm

cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: redhat-operators
  namespace: olm
spec:
  displayName: Red Hat Operators
  grpcPodConfig:
    extractContent:
      cacheDir: /tmp/cache
      catalogDir: /configs
    memoryTarget: 30Mi
    nodeSelector:
      kubernetes.io/os: linux
      node-role.kubernetes.io/master: ""
    priorityClassName: system-cluster-critical
    securityContextConfig: restricted
    tolerations:
    - effect: NoSchedule
      key: node-role.kubernetes.io/master
      operator: Exists
    - effect: NoExecute
      key: node.kubernetes.io/unreachable
      operator: Exists
      tolerationSeconds: 120
    - effect: NoExecute
      key: node.kubernetes.io/not-ready
      operator: Exists
      tolerationSeconds: 120
  icon:
    base64data: ""
    mediatype: ""
  image: registry.redhat.io/redhat/redhat-operator-index:v4.20
  secrets:
  - pull-secret
  priority: -100
  publisher: Red Hat
  sourceType: grpc
  updateStrategy:
    registryPoll:
      interval: 10m
EOF
