# Development Server Onboarding Guide

Status: Approved

## Purpose

This guide describes how to prepare an Ubuntu 24.04 LTS development server for
use with `dev-connect`.

The development server is not part of Kubernetes. SSH authentication and user
authorization remain on the development server through standard OpenSSH.

This guide covers:

- server prerequisites,
- Linux user creation,
- developer SSH key generation,
- `authorized_keys` installation,
- SSH host key collection,
- dev-connect host key inventory update,
- gateway deployment values,
- client configuration.

## Security Model

Do not store SSH private keys, SSH passwords, user databases, or Kubernetes
credentials in Kubernetes.

The gateway forwards TCP only. It does not authenticate SSH users and does not
inspect SSH traffic.

The development server remains responsible for:

- Linux user accounts,
- OpenSSH authentication,
- `authorized_keys`,
- PAM or enterprise identity integration where used,
- shell and sudo policy,
- server-side audit logging.

## Prerequisites

Required:

- Ubuntu 24.04 LTS development server.
- OpenSSH server installed and enabled.
- Private network reachability from the Kubernetes gateway Pods to the server
  on TCP port `22`.
- Administrative access to the development server.
- Platform GitOps repository or approved configuration path for pinned SSH host
  keys.
- Developer workstation with `dev-connect`, OpenSSH client, VS Code Desktop,
  kubeconfig, and Rancher-managed Kubernetes access.

Example values used below:

```text
Target alias: dev01
Development server IP: 172.28.192.14
Linux user: anwendt
Client private key: ~/.ssh/id_rsa-t-systems
Kubernetes namespace: dev-connect
Gateway release name: dev-connect-dev01
Gateway service name: dev-connect-gateway-dev01
```

Adjust these values for the target environment.

## 1. Prepare the Development Server

Run on the development server as an administrator:

```text
sudo apt-get update
sudo apt-get install -y openssh-server
sudo systemctl enable --now ssh
sudo systemctl status ssh --no-pager
```

Confirm SSH listens on TCP port `22`:

```text
sudo ss -tlnp | grep ':22'
```

Recommended OpenSSH baseline in `/etc/ssh/sshd_config`:

```text
PubkeyAuthentication yes
PasswordAuthentication no
PermitRootLogin no
```

Apply changes if the file was modified:

```text
sudo sshd -t
sudo systemctl reload ssh
```

## 2. Create the Linux User

Create the developer user on the development server:

```text
sudo adduser --disabled-password --gecos "" anwendt
```

Optional sudo access, only if required by the development use case:

```text
sudo usermod -aG sudo anwendt
```

If sudo access is not required, do not add the user to `sudo`.

## 3. Generate the Developer SSH Key

Run on the developer workstation, not on the server:

```text
ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa-t-systems -C "anwendt@dev01"
```

Use a passphrase unless the enterprise uses a hardware-backed or otherwise
approved key-management flow.

The private key stays on the developer workstation:

```text
~/.ssh/id_rsa-t-systems
```

The public key is:

```text
~/.ssh/id_rsa-t-systems.pub
```

## 4. Install the Developer Public Key

Copy the public key to the development server through an approved administrative
path.

Example with administrative shell access on the server:

```text
sudo install -d -m 700 -o anwendt -g anwendt /home/anwendt/.ssh
cat ~/.ssh/id_rsa-t-systems.pub | sudo tee -a /home/anwendt/.ssh/authorized_keys >/dev/null
sudo chown anwendt:anwendt /home/anwendt/.ssh/authorized_keys
sudo chmod 600 /home/anwendt/.ssh/authorized_keys
```

If the public key file is not available on the server, paste the public key
content into `/home/anwendt/.ssh/authorized_keys` through the approved
administrative process.

Never copy the private key to the development server.

## 5. Collect the Server Host Key

`dev-connect` requires SSH host key pinning. The host key identifies the server;
it is not the developer's private key.

Preferred command from a trusted administrative network path:

```text
ssh-keyscan -t ed25519 172.28.192.14
```

Example output:

```text
172.28.192.14 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExamplePinnedHostKeyReplaceFromGitOpsInventory
```

Normalize the host key inventory entry to the dev-connect target alias:

```text
dev01 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExamplePinnedHostKeyReplaceFromGitOpsInventory dev01
```

Validate the key through the approved platform process before publishing it.
Do not use trust-on-first-use for production.

## 6. Update the Host Key Inventory

Add or update the host key in the Platform GitOps repository or approved
configuration source.

Client configuration shape:

```yaml
hostKeys:
  dev01: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExamplePinnedHostKeyReplaceFromGitOpsInventory dev01
```

If the key inventory uses a different reference name, configure `hostKeyRef` on
the target:

```yaml
targets:
  dev01:
    gateway: dev01
    user: anwendt
    identityFile: /Users/anwendt/.ssh/id_rsa-t-systems
    hostKeyRef: ubuntu-dev01-ed25519
hostKeys:
  ubuntu-dev01-ed25519: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExamplePinnedHostKeyReplaceFromGitOpsInventory dev01
```

When `hostKeyRef` is omitted, `dev-connect` uses the target name as the host key
reference.

## 7. Deploy the Gateway for the Server

Deploy one gateway release for the target development server.

Example:

```text
helm upgrade --install dev-connect-dev01 oci://ghcr.io/anwendt/charts/dev-connect-gateway \
  --namespace dev-connect \
  --create-namespace \
  --set target.name=dev01 \
  --set target.host=172.28.192.14 \
  --set target.port=22 \
  --set networkPolicy.backendCIDR=172.28.192.14/32 \
  --set networkPolicy.dnsEgress.enabled=false \
  --set monitoring.enabled=true \
  --wait \
  --timeout 5m
```

Expected result:

- Service is `ClusterIP`.
- Service exposes TCP `22`.
- Gateway Pod listens internally on TCP `2222`.
- NetworkPolicy allows egress only to the approved development server on TCP
  `22`, plus DNS only when DNS-based backend addressing is configured.

## 8. Create the Client Configuration

Example macOS/Linux client configuration:

```yaml
apiVersion: dev-connect/v1
kind: DevConnectConfig
contexts:
  central-dev:
    cluster: central-dev
    gateway: dev01
clusters:
  central-dev:
    kubeconfig: /Users/anwendt/.kube/central-dev-cluster.yaml
    kubernetesContext: ""
    proxy:
      enabled: false
gateways:
  dev01:
    namespace: dev-connect
    service: dev-connect-gateway-dev01
    port: 22
targets:
  dev01:
    gateway: dev01
    user: anwendt
    identityFile: /Users/anwendt/.ssh/id_rsa-t-systems
hostKeys:
  dev01: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExamplePinnedHostKeyReplaceFromGitOpsInventory dev01
ssh:
  manageUserConfig: true
  userConfigPath: ""
vscode:
  launcherPath: ""
  isolatedUserDataDir: false
```

Example Windows path values:

```yaml
clusters:
  central-dev:
    kubeconfig: 'C:\Users\anwendt\.kube\central-dev-cluster.yaml'
    kubectlPath: 'C:\Program Files\dev-connect\kubectl.exe'
targets:
  dev01:
    gateway: dev01
    user: anwendt
    identityFile: 'C:\Users\anwendt\.ssh\id_rsa-t-systems'
```

Use single quotes for Windows paths in YAML.

## 9. Configure Remote VS Code Server Proxy

Some VS Code Remote SSH extensions execute on the remote development server.
Those remote extension processes must use a proxy that is reachable from the
development server, not necessarily the proxy used by the developer workstation.

This is especially relevant for GitHub Copilot or other extensions that perform
network calls from the VS Code Server side.

`dev-connect` can write VS Code Server proxy settings after the tunnel is ready
and before VS Code is launched.

Example:

```yaml
ssh:
  manageUserConfig: true
  userConfigPath: ""
vscode:
  launcherPath: ""
  isolatedUserDataDir: false
  remoteSetup:
    enabled: true
    sshPath: ""
    httpProxy: http://remote-proxy.example.corp:8080
    httpsProxy: http://remote-proxy.example.corp:8080
    noProxy: localhost,127.0.0.1,0.0.0.0,10.0.0.0/8,.svc,.cluster.local
    proxySupport: override
    batchMode: true
```

When `vscode.remoteSetup.enabled: true`, `dev-connect connect` writes these
files on the remote development server:

```text
~/.vscode-server/server-env-setup
~/.vscode-server/data/Machine/settings.json
```

The generated remote environment exports:

```text
HTTP_PROXY
HTTPS_PROXY
NO_PROXY
http_proxy
https_proxy
no_proxy
```

The generated VS Code Server machine settings include:

```json
{
  "http.proxy": "http://remote-proxy.example.corp:8080",
  "http.proxySupport": "override",
  "http.noProxy": [
    "localhost",
    "127.0.0.1",
    "0.0.0.0",
    "10.0.0.0/8",
    ".svc",
    ".cluster.local"
  ]
}
```

For Windows clients using Git for Windows SSH, set `sshPath` to the Git SSH
binary when the default Windows OpenSSH client does not use the expected SSH
agent:

```yaml
vscode:
  isolatedUserDataDir: false
  remoteSetup:
    enabled: true
    sshPath: 'C:\Users\anwendt\AppData\Local\Programs\Git\usr\bin\ssh.exe'
    httpProxy: http://remote-proxy.example.corp:8080
    httpsProxy: http://remote-proxy.example.corp:8080
    noProxy: localhost,127.0.0.1,0.0.0.0,10.0.0.0/8,.svc,.cluster.local
    proxySupport: override
    batchMode: true
```

Use `batchMode: true` when the key is already available through the selected
SSH client and agent. Set `batchMode: false` only when the setup SSH command
must prompt interactively for a password or key passphrase.

The remote proxy values are independent from `clusters.<name>.proxy`.

- `clusters.<name>.proxy` affects only local `kubectl` child processes.
- `vscode.remoteSetup` affects VS Code Server files on the remote development
  server.

Do not put proxy credentials into the client configuration unless this is
explicitly approved by the enterprise secret-handling policy.

## 10. Validate the Client Configuration

Run on the developer workstation:

```text
dev-connect --config /path/to/dev01.yaml config validate
dev-connect --config /path/to/dev01.yaml list
```

Run a tunnel-only smoke test:

```text
dev-connect --config /path/to/dev01.yaml connect dev01 --no-code --output json
dev-connect --config /path/to/dev01.yaml status --output json
dev-connect --config /path/to/dev01.yaml disconnect --output json
```

Run the full VS Code flow:

```text
dev-connect --config /path/to/dev01.yaml connect dev01
```

Expected result:

- Kubernetes preflight succeeds through `kubectl`.
- `kubectl port-forward` remains running.
- Temporary `known_hosts` contains the pinned server host key.
- OpenSSH authenticates as `anwendt` on the development server using the
  configured private key.
- VS Code Desktop opens Remote SSH target `dev01`.
- VS Code Server installs or updates on the target server.
- If `vscode.remoteSetup.enabled` is configured, remote VS Code Server proxy
  files are written before VS Code launches.

## 11. Troubleshooting Checks

Check SSH authentication directly through the generated dev-connect session
config while a session is active:

```text
ssh -F "$HOME/Library/Application Support/dev-connect/session/ssh/ssh_config" dev01
```

Linux:

```text
ssh -F "$HOME/.config/dev-connect/session/ssh/ssh_config" dev01
```

Windows PowerShell:

```text
ssh -F "$env:APPDATA\dev-connect\session\ssh\ssh_config" dev01
```

Common failures:

- `Permission denied (publickey)`: the user's public key is missing or wrong in
  `authorized_keys`, the private key path is wrong, or server-side SSH policy
  rejects the key.
- `Host key verification failed`: the pinned `hostKeys` entry does not match the
  server host key.
- `connect: connection refused`: OpenSSH is not listening on the target server
  or the gateway cannot reach TCP port `22`.
- `kubectl exited with code 1`: kubeconfig, Rancher authentication, proxy, or
  RBAC must be corrected before SSH is attempted.

Do not disable `StrictHostKeyChecking`.
