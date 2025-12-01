#!/usr/bin/env bash

# This script must be run after installing the hosted cluster.
# This is required for Velero and Node Agent to run but then, HostedCluster on AKS fails to start due to this.
oc apply -f https://raw.githubusercontent.com/openshift/api/refs/heads/master/security/v1/zz_generated.crd-manifests/0000_03_config-operator_01_securitycontextconstraints.crd.yaml

oc apply -f https://raw.githubusercontent.com/openshift/api/refs/heads/master/route/v1/zz_generated.crd-manifests/routes.crd.yaml

oc apply -f https://raw.githubusercontent.com/openshift/api/refs/heads/master/config/v1/zz_generated.crd-manifests/0000_10_config-operator_01_infrastructures-Default.crd.yaml 

cat <<EOF | oc apply -f -
apiVersion: config.openshift.io/v1
kind: Infrastructure
metadata:
  labels:
    hypershift.openshift.io/managed: "true"
  name: cluster
spec:
  cloudConfig:
    name: ""
  platformSpec:
    azure: {}
    type: Azure
EOF
