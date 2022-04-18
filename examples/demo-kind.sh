#!/usr/bin/env bash

set -euo pipefail

script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
repo_dir="${script_dir}/.."

# Create local Kubernetes in Docker cluster
kind create cluster \
    --image kindest/node:v1.23.5 \
    --name lastpass

# Install the Secrets Store CSI Driver as described in
# https://secrets-store-csi-driver.sigs.k8s.io/getting-started/installation.html
helm repo add secrets-store-csi-driver https://kubernetes-sigs.github.io/secrets-store-csi-driver/charts
helm install csi-secrets-store secrets-store-csi-driver/secrets-store-csi-driver \
    --namespace kube-system \
    --version "1.0.0" \
    --set "syncSecret.enabled=true" \
    --set "enableSecretRotation=true" \
    --wait
kubectl -n kube-system wait --for=condition=Ready pods -l "app=secrets-store-csi-driver" --timeout=5m

# Install the LastPass provider.
kubectl apply -f "${repo_dir}/deployment/lastpass-provider.yml"
kubectl -n kube-system wait --for=condition=Ready pods -l "app=secrets-store-csi-driver-provider-lastpass" --timeout=5m

# Deploy a Pod getting secrets from LastPass.
# Environment variables LASTPASS_USERNAME and LASTPASS_MASTERPASSWORD must be exported.
sed "s/((lastpass-username))/${LASTPASS_USERNAME}/; s/((lastpass-masterpassword))/${LASTPASS_MASTERPASSWORD}/" \
    "${repo_dir}"/examples/*.yml 2>/dev/null | kubectl apply -f -
kubectl wait --for=condition=Ready pod/mypod --timeout=5m

# Print the mounted LastPass file names.
kubectl exec mypod -- ls /mnt/secrets-store/
