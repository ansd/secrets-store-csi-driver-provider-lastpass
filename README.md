# LastPass Provider for Secrets Store CSI Driver

An experimental [LastPass](https://lastpass.com) provider for [Kubernetes Secrets Store CSI Driver](https://secrets-store-csi-driver.sigs.k8s.io/).

## Background
As explained in [this](https://youtu.be/w0k7MI6sCJg?t=177) KubeCon talk, there 4 different approaches in K8s to consume secrets from external secret stores:
1. Direct API
2. Controller to mirrors secrets in K8s
3. Sidecar + MutatingWebhookConfiguration
4. Secrets Store CSI Driver

This repo implements the last approach for the external secret store `LastPass`.

## Features
* Mount any LastPass items (e.g. passwords, secure notes, SSH keys) into Pods as a Volume.
* Sync as K8s Secret object: Mirror the mounted LastPass items into K8s Secret objects.
* Auto rotation: When items are updated in LastPass after the initial pod deployment, they will be periodically updated in the pod mount and the K8s Secret objects.

## Usage

1. Define the items you want to fetch from LastPass in [examples/secret-provider-class.yml](https://github.com/ansd/secrets-store-csi-driver-provider-lastpass/blob/85ebeb8bd13155577cb6c5b679274edba20652cc/examples/secret-provider-class.yml#L10-L16).
2. `LASTPASS_USERNAME=<my-user> LASTPASS_MASTERPASSWORD=<my-password> ./examples/demo-kind.sh`

## Design

Comparing the design of this LastPass provider to
* [Vault Provider](https://github.com/hashicorp/vault-csi-provider)
* [AWS Provider](https://github.com/aws/secrets-store-csi-driver-provider-aws)
* [GCP Provider](https://github.com/GoogleCloudPlatform/secrets-store-csi-driver-provider-gcp)
* [Azure Provider](https://github.com/Azure/secrets-store-csi-driver-provider-azure/)

it resembles most [Service Principle Mode in Azure Key Vault](https://azure.github.io/secrets-store-csi-driver-provider-azure/configurations/identity-access-modes/service-principal-mode/) meaning
1. LastPass username and master password (to authenticate to LastPass) need to be created as a K8s Secret which is stored (by default) as plaintext in etcd.
2. Username and master password is the only supported way to connect to LastPass because LastPass does not support authentication using a K8s Service Account Token.

## Troubleshooting
If you have issues running above example, run
```bash
kubectl -n default describe pods/mypod
kubectl -n kube-system logs -l app=secrets-store-csi-driver-provider-lastpass
kubectl -n kube-system logs -l app.kubernetes.io/instance=csi-secrets-store
```
If logs show
>failed to authenticate with LastPass server" err="unifiedloginresult: Try again OR look for an email from LastPass to verify it's you."

check your inbox for e-mails with the subject `LastPass Verification Email` to verify the new location or device.

You can disable e-mail verification in LastPass Chrome extension `Account Settings` -> `General` -> `Show Advanced Settings` -> `Disable Email Verification`.

## Notes

* Although two-factor authentication works, it should be disabled (unless you want to manually approve push notifications).
* There is a paid plan called "LastPass Identity". With this plan, you can create a separate machine user and assign this user an [IP policy](https://www.lastpass.com/policies/ip-address) that allowlists the public IP addresses of the K8s nodes blocking all other IP addresses by default. You can also allowlist the DNS name of the K8s nodes (based on reverse DNS lookup of their IP addresses). This adds security given that two-factor authentication is disabled.
