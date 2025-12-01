#!/usr/bin/env bash

set -euox pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

export OADP_AZURE_CREDS_FILE=oadp/azure-oadp-credentials
export AZURE_STORAGE_ACCOUNT="mgencur"
export AZ_CLIENT_SECRET=$(az ad sp create-for-rbac --name "mgencur-oadp" --role "Contributor" --query 'password' --scopes "/subscriptions/$SUBSCRIPTION_ID" -o tsv --only-show-errors)
export AZ_CLIENT_ID=$(az ad sp list --display-name "mgencur-oadp" --query '[0].appId' -o tsv)
# Storage manually created in advance
export AZ_STORAGE_ACCOUNT_KEY=$(az storage account keys list --account-name ${AZURE_STORAGE_ACCOUNT} --resource-group $PERSISTENT_RG_NAME --query '[0].value' -o tsv)

"${SCRIPT_DIR}/oadp/00_deploy-olm.sh"
"${SCRIPT_DIR}/oadp/01_label-nodes.sh"
"${SCRIPT_DIR}/oadp/03_deploy-catalog-source.sh"
"${SCRIPT_DIR}/oadp/04_deploy-oadp-operator.sh"
"${SCRIPT_DIR}/oadp/05_create-pull-secret-adp.sh"
"${SCRIPT_DIR}/oadp/06_patch-sa.sh"
"${SCRIPT_DIR}/oadp/07_add-api-sesources.sh"
"${SCRIPT_DIR}/oadp/08_setup-azure-creds.sh"
"${SCRIPT_DIR}/oadp/09_setup-dpa.sh"
