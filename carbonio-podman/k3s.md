# Carbonio on k3s

Same chart as the podman setup — `platform` defaults to `k3s` instead of
`podman`, so Services, imagePullSecrets and PVCs render automatically.

> **On macOS, jump to [§7](#7-macos-k3d)** — it is a self-contained
> walkthrough. §1 is Linux-only; the rest of this page applies to both.
>
> **One-shot bootstrap (Linux):** `./install_k3s.sh` runs §1 + §3 in one
> command. Helm is OS-dependent so the script does not install it; install Helm
> before §4 per [helm.sh/docs/intro/install](https://helm.sh/docs/intro/install/).
> The manual commands below stay as the underlying reference.
>
> Or use the `Makefile`: `make help` lists every target. The k3s lifecycle
> (`install_k3s`, `start_k3s`, `stop_k3s`, `uninstall_k3s`) comes from
> `Makefile.linux` or `Makefile.darwin`, picked from `uname`; everything else
> (`install_k3s_full`, `install_portainer`, `install_chartmuseum`,
> `carbonio_install`, `carbonio_upgrade`, `carbonio_uninstall`,
> `carbonio_purge`, `carbonio_provision`) is shared.

## 1. Install k3s (single-node VM, Linux)
```bash
# Disable Traefik + ServiceLB so they don't grab host ports 80/443 — composed-ui
# binds 443 via hostPort. Make the kubeconfig world-readable so helm/kubectl
# don't need sudo.
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--disable traefik --disable servicelb --write-kubeconfig-mode=644" sh -
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml   # add to ~/.bashrc to persist
```
k3s ships the `local-path` StorageClass the chart's PVCs use — no storage setup
needed.

To stop it:
```bash
sudo /usr/local/bin/k3s-killall.sh
```
It will stop k3s and the workload. If you just stop k3s the containers will 
stay up.

## 2. Install Helm
```bash
curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

## 3. Namespace + private-registry pull secret
```bash
kubectl create namespace carbonio
kubectl -n carbonio create secret docker-registry zextras-registry \
  --docker-server=registry.dev.zextras.com \
  --docker-username=<user> --docker-password=<pass>
```
(Or set `image.pullSecret.create=true` with the credentials in values and skip
this.) `install_k3s.sh` creates the secret only when both `REGISTRY_USER` and
`REGISTRY_PASS` are set; otherwise create it before installing the chart.

## Terminal UI (k9s) — recommended

The fastest way to watch a Carbonio install: every pod, container, log, exec
shell and port-forward from one keyboard-driven view, no extra pod in the
cluster. It reads the same `~/.kube/config` as kubectl.

```bash
brew install k9s          # macOS; Linux: https://k9scli.io/topics/install/
k9s -n carbonio
```

Worth knowing: `Enter` expands a pod into its containers (mailbox has 11),
`l` logs, `s` shell, `d` describe, `Shift-F` port-forward, `Ctrl-D` delete,
`0` all namespaces, `:jobs` / `:pvc` / `:svc` to switch view, `Ctrl-C` quit.

## Web UI (Portainer) — recommended

Gives you a browser view of every pod, container, logs, and an exec console —
the main reason to bother with k3s over plain podman for day-to-day dev.
Install via NodePort (a LoadBalancer service would stay `pending` since
`servicelb` is disabled):

```bash
helm repo add portainer https://portainer.github.io/k8s/
helm repo update
helm install portainer portainer/portainer -n portainer --create-namespace \
  --set service.type=NodePort
```
Open `https://<vm-ip>:30779` (or `http://<vm-ip>:30777`) and set the admin
password on first load — Portainer auto-detects the local cluster and lists
every namespace, pod, container, their logs, and an exec console.

If the NodePort is firewalled, tunnel instead:
```bash
kubectl -n portainer port-forward svc/portainer 9443:9443   # then https://localhost:9443
```

Optionally patch portainer to bind to 9443:
```bash
kubectl -n portainer patch deploy portainer --type=json -p='[
    {"op":"add","path":"/spec/template/spec/containers/0/ports/1/hostPort","value":9443}
  ]'
```

## Helm registry — optional

Skip this if you have the repo checked out — `helm install carbonio chart/`
from the source directory works fine. Use these only when:

- **Zot (OCI push)** — you want `helm install` from CI or another machine
  without cloning the repo.
- **ChartMuseum** — you want to install/upgrade the chart from Portainer's UI.
  Portainer CE supports only classic HTTP Helm repos (OCI is a Business
  Edition feature), so Zot alone won't show up in Portainer's Helm tab.
  ChartMuseum is a tiny in-cluster HTTP repo that bridges the gap.

### Zot (OCI)
```bash
helm registry login registry.dev.zextras.com    # one-time
CHART_VERSION=$(awk '$1 == "version:" {print $2}' chart/Chart.yaml)
helm package chart/
helm push "carbonio-${CHART_VERSION}.tgz" oci://registry.dev.zextras.com/dev
```

### ChartMuseum
Install (one-time):
```bash
helm repo add chartmuseum-charts https://chartmuseum.github.io/charts
helm repo update
helm install chartmuseum chartmuseum-charts/chartmuseum \
  -n chartmuseum --create-namespace \
  --set env.open.DISABLE_API=false \
  --set env.open.ALLOW_OVERWRITE=true \
  --set persistence.enabled=true \
  --set service.type=NodePort \
  --set service.nodePort=30808
```

On macOS, `<vm-ip>` below is `localhost` only if 30808 was published when the
k3d cluster was created (`EXTRA_PORTS`, §7); otherwise port-forward it.

Push the chart (bump `chart/Chart.yaml` `version:` first):
```bash
CHART_VERSION=$(awk '$1 == "version:" {print $2}' chart/Chart.yaml)
helm package chart/
curl --data-binary "@carbonio-${CHART_VERSION}.tgz" http://<vm-ip>:30808/api/charts
```

For local Helm CLI access, add the deployed repository under its own alias:
```bash
helm repo add carbonio-local http://<vm-ip>:30808
helm repo update carbonio-local
```

In Portainer → **Settings → Helm**, add repository URL
`http://chartmuseum.chartmuseum:8080` (in-cluster ClusterIP DNS — Portainer is
itself a pod). The chart then appears under **Helm → Available** with Install
/ Upgrade / Uninstall buttons.

## 4. Install the chart

> **On an arm64 node (Apple Silicon), enable Rosetta.** Most `:devel` tags are
> multi-arch and containerd picks the node platform, but a handful of images
> (message-broker, message-dispatcher, videoserver, tasks) ship amd64 only and
> run emulated.

From the local checkout, use the Make target; its 45-minute timeout covers cold
image pulls, mailbox startup, automatic provisioning, and the first mailbox
restart:

```bash
make carbonio_install
# equivalent:
helm install carbonio chart/ -n carbonio --timeout 45m
```

Add `-f chart/values.local.yaml` to the raw Helm command to select groups or
other settings; see the documented options in [README.md](README.md) and
`chart/values.yaml`.

If you published the chart in the optional registry section instead:

```bash
# Zot (set this to the version you published):
CHART_VERSION=0.1.3
helm install carbonio oci://registry.dev.zextras.com/dev/carbonio \
  --version "$CHART_VERSION" -n carbonio --timeout 45m

# ChartMuseum, after adding the carbonio-local alias above:
helm install carbonio carbonio-local/carbonio -n carbonio --timeout 45m
```

Portainer users can instead select the Carbonio chart under
**Applications → Create from code → Helm chart**.

Watch pods, then uninstall when done:

```bash
kubectl -n carbonio get pods -w
make carbonio_uninstall
```

## 5. Provisioning

On k3s, `provision.auto: true` enables a post-install/post-upgrade Helm hook by
default. It waits for `carbonio-mailbox-0`, runs `chart/files/zmprov.prov`,
restarts the mailbox after first provisioning, and refreshes composed-ui's
nginx configuration. The hook renders when `groups.mails` is enabled and also
assumes `groups.ui` is enabled; set `provision.auto=false` when running mails
without the UI. The hook is idempotent and contributes to the Helm command
timeout.

Use the manual script only as a fallback, or when installing with
`--set provision.auto=false`:

```bash
./provision.sh
# override defaults if needed:
NAMESPACE=carbonio PROV_FILE=chart/files/zmprov.prov ./provision.sh
```

If mailbox readiness polling expires, the current manual script may still print
`Done`; confirm its output or verify the domain before treating it as successful:

```bash
kubectl -n carbonio exec carbonio-mailbox-0 -c app -- \
  sh -c 'echo "gd carbonio.localhost" | zmprov'
```

Inspect the automatic hook with:

```bash
kubectl -n carbonio logs job/carbonio-provision
```

## 6. Access

`<vm-ip>` is the k3s node's address on Linux and plain `localhost` on macOS/k3d.

- Web UI: `https://<vm-ip>` (composed-ui, hostPort 443) — self-signed cert
- Admin console: `https://<vm-ip>:6071`
- Consul UI: `http://<vm-ip>:8500`
- Grafana: `http://<vm-ip>:3000` when `monitoring.enabled=true`
- Portainer: `https://<vm-ip>:30779` when installed

On macOS every one of these is `localhost` — but only because the k3d node
container publishes those ports to the Mac (docker `-p`, not `kubectl
port-forward`). A port that is not in the cluster's publish list is simply not
there: Grafana's 3000 and ChartMuseum's 30808 need `EXTRA_PORTS` at cluster
creation (§7), or a one-off `kubectl -n <ns> port-forward svc/<svc> <port>`.

## 7. macOS (k3d)

k3s has no macOS build, so `make install_k3s` creates a **k3d** cluster: the
same k3s, running inside a Docker container. Docker Desktop's own Kubernetes is
*not* used (see "Why not Docker Desktop's Kubernetes" below) — only its docker
daemon, and any daemon will do, since k3d follows `DOCKER_CONTEXT` /
`DOCKER_HOST`.

The node container publishes the chart's hostPorts onto macOS and the pods bind
in that container's network namespace, so the §6 addresses work on `localhost`
with no extra Service — the same exposure model as Linux.

### Before you start

- **Docker Desktop running**, with Rosetta enabled (Settings → General) for the
  amd64-only images, and memory raised to **16 GB** (Settings → Resources); the
  8 GB default is thin for ~19 pods / ~41 containers.
- **helm and kubectl**: `brew install helm kubernetes-cli`. `k3d` is installed
  by `make install_k3s` if missing; `k9s` is optional but worth it.
- **`docker login registry.dev.zextras.com`** with your Zextras account (the
  username is your e-mail address, not the short handle). `make install_k3s`
  reads that credential back out of the macOS keychain to build the pull secret
  — k3s's containerd cannot read `~/.docker/config.json` itself. Export
  `REGISTRY_USER` / `REGISTRY_PASS` instead if you'd rather not log in.

### Step by step

```bash
cd carbonio-podman

make install_k3s        # k3d cluster + namespace + pull secret
make carbonio_install   # helm install; provisioning runs inside it, 45m timeout
k9s -n carbonio         # watch it come up (or: kubectl -n carbonio get pods -w)
```

Then open `https://localhost` (accept the self-signed cert) and log in;
`https://localhost:6071` is the admin console. Everything in §6 applies with
`<vm-ip>` = `localhost`.

The first install is slow — kubelet serializes image pulls, so the 11-container
mailbox alone costs ~11 minutes. Pods that crash before their dependencies are
up (composed-ui needs LDAP for `zmproxyconfgen`) recover on their own restart:
a noisy first few minutes is normal. Later installs reuse the cached images.

### Published ports

The node container publishes a fixed list at creation time: 443 and 6071
(composed-ui), 8500 (consul), 20025 (postfix), 30777/30779 (portainer). Docker
holds each one on macOS whether or not something is behind it, so anything
off by default is opt-in:

```bash
make install_k3s EXTRA_PORTS="3000:3000 30808:30808"   # grafana, chartmuseum
```

Ports cannot be added to a running cluster — `make uninstall_k3s install_k3s
EXTRA_PORTS="..."` to change the list.

### Day to day

| command | what it does |
|---|---|
| `make stop_k3s` / `make start_k3s` | stop and restart the cluster containers; data and images survive |
| `make carbonio_upgrade` | re-apply the chart after editing it |
| `make import IMAGE=localhost/foo:dev` | load a locally built image into the cluster's containerd — k3s does not share the docker image store |
| `k9s` | see the k9s section above |

To wipe things, see §8.

### Why not Docker Desktop's Kubernetes

The chart itself is fully compatible with Docker Desktop's Kubernetes — every
rendered object passes `kubectl apply --dry-run=server` against its apiserver,
with no CRDs and nothing outside `v1`/`apps/v1`/`batch/v1`/`rbac/v1`. What does
not work is the **host exposure model**: Docker Desktop never forwards pod
`hostPort` to macOS, in either provisioning mode. The port binds correctly
inside the VM (a `hostNetwork` pod reaches `127.0.0.1:443`) but is unreachable
from macOS on both `localhost` and the node IP, for TCP and UDP alike. Every
address in §6 is therefore dead on Docker Desktop.

Measured on k8s v1.36.1:

| path | kubeadm | kind |
|---|---|---|
| `hostPort` TCP / UDP | ✗ | ✗ |
| `NodePort` | ✓ | ✗ |
| `LoadBalancer` TCP on `localhost:<port>` | ✓ | ✓ |
| `LoadBalancer` UDP | ✓ | ✗ |
| nodes | 1 | 8 (control-plane + 7 workers) |
| runtime | cri-dockerd — shares the docker image store | containerd in `kindest/node` |
| default StorageClass | `hostpath` (Immediate) | `standard` (WaitForFirstConsumer) |

**Use kubeadm, not kind** (Settings → Kubernetes → Cluster provisioning method).
In kind mode the node containers publish nothing but the apiserver, so both
`hostPort` and `NodePort` die; `LoadBalancer` survives only because
cloud-provider-kind spawns a `kindccm-*` proxy container that publishes the
port, and its UDP publish does not deliver. kind also spreads pods over 8 nodes,
which breaks the RWO PVCs, and its containerd is a separate image store so every
`localhost/*` image needs loading per node. Under kubeadm, cri-dockerd shares
the docker image store, so a locally built `localhost/*` image is visible to
Kubernetes with no import step (on k3s the same image needs
`k3s ctr images import`).

The chart therefore ships no LoadBalancer Service: kubeadm was the only place one
helped, and on both k3s paths `servicelb` is disabled, where it would sit at
`EXTERNAL-IP: <pending>`. To use Docker Desktop's kubeadm cluster anyway, add a
`type: LoadBalancer` Service selecting `app: carbonio-composed-ui` by hand — it
gets `EXTERNAL-IP: localhost` (verified serving the login page and a successful
SOAP `AuthRequest`), and UDP works under kubeadm. For one-off debugging,
`kubectl port-forward` is enough, but it is TCP-only and needs root to bind 443.

## 8. Uninstall, reset, start over

Removing the release (both platforms):

```bash
make carbonio_uninstall   # helm uninstall + drop the leftover provision Job,
                          # then wait for every pod to actually be gone
make carbonio_purge       # the same, plus delete the PVCs
```

Use `carbonio_purge` whenever you want the next install to be a real first
install: keeping the volumes makes the provision hook skip with "Domain already
exists", and a reused openldap volume can come back with indexes that answer
nothing — equality searches return 0 while slapd looks perfectly healthy.

Don't run a bare `helm uninstall`. composed-ui and postfix are plain Pods, so a
reinstall that races their termination gets them adopted by Helm, which then
reports success and leaves nothing to recreate them. Both targets above wait the
termination out for that reason.

Throwing away the cluster itself:

```bash
make uninstall_k3s        # macOS: k3d cluster delete — volumes and image cache go too
                          # Linux: k3s-uninstall.sh
```

On macOS this is the nuclear option rather than the quick one: the cached images
live in the node container, so the next install re-pulls all of them. Prefer
`make carbonio_purge` for a clean slate, and keep `uninstall_k3s` for changing
the published port list or a genuinely broken cluster. To just get the laptop's
resources back, `make stop_k3s`.
