#!/usr/bin/env bash

curl -L https://github.com/operator-framework/operator-lifecycle-manager/releases/download/v0.31.0/install.sh -o install.sh && chmod +x install.sh

./install.sh v0.31.0
