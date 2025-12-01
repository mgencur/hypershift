#!/usr/bin/env bash

set -euox pipefail

## Create the DPA
cat <<EOF > dpa.yaml
apiVersion: oadp.openshift.io/v1alpha1
kind: DataProtectionApplication
metadata:
  name: dpa-azure
  namespace: openshift-adp
spec:
  backupLocations:
    - velero:
        config:
          resourceGroup: ${PERSISTENT_RG_NAME}
          storageAccount: ${AZURE_STORAGE_ACCOUNT}
          subscriptionId: ${SUBSCRIPTION_ID}
          storageAccountKeyEnvVar: AZURE_STORAGE_ACCOUNT_ACCESS_KEY
        credential:
          key: cloud
          name: cloud-credentials-azure
        provider: azure
        default: true
        objectStorage:
          bucket: oadp
          prefix: backup-objects
  snapshotLocations:
    - velero:
        config:
          resourceGroup: ${PERSISTENT_RG_NAME}
          subscriptionId: ${SUBSCRIPTION_ID}
          incremental: "true"
        provider: azure
  configuration:
    nodeAgent:
      enable: true
      uploaderType: kopia
    velero:
      defaultPlugins:
        - openshift
        - azure
        - kubevirt
        - csi
      customPlugins:
        - name: hypershift-oadp-plugin
          image: quay.io/redhat-user-workloads/ocp-art-tenant/oadp-hypershift-oadp-plugin-oadp-1-5:oadp-1.5
      resourceTimeout: 2h
EOF

oc apply -f dpa.yaml

until kubectl get serviceaccount velero -n openshift-adp &>/dev/null; do
    echo "Waiting for serviceaccount velero to exist..."
    sleep 5
done

kubectl patch serviceaccount -n openshift-adp velero -p '{"imagePullSecrets": [{"name": "pull-secret"}]}'

if oc get pod -n openshift-adp -l deploy=velero &>/dev/null; then
  oc delete pod -n openshift-adp -l deploy=velero
fi
if oc get pod -n openshift-adp -l name=node-agent &>/dev/null; then
  oc delete pod -n openshift-adp -l name=node-agent
fi

until [[ $(kubectl get pod -l deploy=velero -n openshift-adp -oname | wc -l) != 0 ]]; do
    echo "Waiting for Pod velero to exist..."
    sleep 5
done
kubectl wait --for=condition=Ready pod -l deploy=velero -n openshift-adp --timeout=300s

until [[ $(kubectl get pod -l name=node-agent -n openshift-adp -oname | wc -l) != 0 ]]; do
    echo "Waiting for Pod node-agent to exist..."
    sleep 5
done
kubectl wait --for=condition=Ready pod -l name=node-agent -n openshift-adp --timeout=300s