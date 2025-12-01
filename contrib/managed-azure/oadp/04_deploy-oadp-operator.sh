#!/usr/bin/env bash

set -euox pipefail

cat <<EOF | oc apply -f -
---
apiVersion: v1
kind: Namespace
metadata:
  name: openshift-adp
  labels:
    openshift.io/cluster-monitoring: "true"
  annotations:
    workload.openshift.io/allowed: management
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: redhat-oadp-operator-operatorgroup
  namespace: openshift-adp
spec:
  targetNamespaces:
  - openshift-adp
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: redhat-oadp-operator
  namespace: openshift-adp
spec:
  channel: "stable"
  name: redhat-oadp-operator
  source: redhat-operators
  sourceNamespace: olm
  installPlanApproval: Automatic
EOF