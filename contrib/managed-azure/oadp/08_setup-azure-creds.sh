#!/usr/bin/env bash

set -euox pipefail

## Create the Secret file
cat <<EOF > $OADP_AZURE_CREDS_FILE
AZURE_SUBSCRIPTION_ID=${SUBSCRIPTION_ID}
AZURE_TENANT_ID=${TENANT_ID}
AZURE_CLIENT_ID=${AZ_CLIENT_ID}
AZURE_CLIENT_SECRET=${AZ_CLIENT_SECRET}
AZURE_RESOURCE_GROUP=${PERSISTENT_RG_NAME}
AZURE_STORAGE_ACCOUNT_ACCESS_KEY=${AZ_STORAGE_ACCOUNT_KEY}
EOF

## Create the Secret
oc delete secret cloud-credentials-azure -n openshift-adp || true
oc create secret generic cloud-credentials-azure -n openshift-adp --from-file cloud=$OADP_AZURE_CREDS_FILE
