# LastPass Provider for Secret Store CSI Driver

A very experimental [LastPass](https://lastpass.com) provider for Kubernetes [Secrets Store CSI Driver](https://secrets-store-csi-driver.sigs.k8s.io/).

## Usage

1. Define the items you want to fetch from LastPass in [examples/secret-provider-class.yml](https://github.com/ansd/secrets-store-csi-driver-provider-lastpass/blob/f7368824aa7ebf85f76c0aa512f3b710da40dc90/examples/secret-provider-class.yml#L13-L22).
2. `LASTPASS_USERNAME=<my-user> LASTPASS_MASTERPASSWORD=<my-password> ./examples/demo-kind.sh`
