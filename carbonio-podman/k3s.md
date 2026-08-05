# Carbonio on k3s

Same chart as the podman setup — `platform` defaults to `k3s` instead of
`podman`, so Services, imagePullSecrets and PVCs render automatically.

> **One-shot bootstrap:** `./install_k3s.sh` runs §1 + §3 (and optionally
> Portainer / ChartMuseum) in one command. Helm is OS-dependent so the script
> does not install it; install Helm before §4, or before using
> `install_k3s_full`, per [helm.sh/docs/intro/install](https://helm.sh/docs/intro/install/).
> The manual commands below stay as the underlying reference.
>
> Or use the `Makefile`: `make help` lists every target
> (`install_k3s`, `install_k3s_full`, `start_k3s`, `stop_k3s`,
> `uninstall_k3s`, `carbonio_install`, `carbonio_upgrade`,
> `carbonio_uninstall`, `carbonio_provision`).

## 1. Install k3s (single-node VM)
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

- Web UI: `https://<vm-ip>` (composed-ui, hostPort 443)
- Consul UI: `http://<vm-ip>:8500`
- Grafana: `http://<vm-ip>:3000` when `monitoring.enabled=true`


