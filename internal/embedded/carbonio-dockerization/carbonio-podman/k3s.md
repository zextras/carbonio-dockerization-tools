# Carbonio on k3s

Same chart as the podman setup — `platform` defaults to `k3s` instead of
`podman`, so Services, imagePullSecrets and PVCs render automatically.

> **One-shot bootstrap:** `./install_k3s.sh` runs §1 + §3 (and optionally
> Portainer / ChartMuseum) in one command. Helm is OS-dependent so it's left
> to you — install it before §4 per [helm.sh/docs/intro/install](https://helm.sh/docs/intro/install/).
> The manual commands below stay as the underlying reference.
>
> Or use the `Makefile`: `make help` lists every target
> (`install_k3s`, `start_k3s`, `stop_k3s`, `uninstall_k3s`, `carbonio_install`,
> `carbonio_upgrade`, `carbonio_uninstall`, `carbonio_provision`).

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
this.)

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
helm package chart/                             # → carbonio-0.1.0.tgz
helm push carbonio-0.1.0.tgz oci://registry.dev.zextras.com/dev
```

### ChartMuseum
Install (one-time):
```bash
helm repo add chartmuseum https://chartmuseum.github.io/charts
helm repo update
helm install chartmuseum chartmuseum/chartmuseum \
  -n chartmuseum --create-namespace \
  --set env.open.DISABLE_API=false \
  --set env.open.ALLOW_OVERWRITE=true \
  --set persistence.enabled=true \
  --set service.type=NodePort \
  --set service.nodePort=30808
```

Push the chart (bump `chart/Chart.yaml` `version:` first):
```bash
helm package chart/
curl --data-binary "@carbonio-0.1.0.tgz" http://<vm-ip>:30808/api/charts
```

In Portainer → **Settings → Helm**, add repository URL
`http://chartmuseum.chartmuseum:8080` (in-cluster ClusterIP DNS — Portainer is
itself a pod). The chart then appears under **Helm → Available** with Install
/ Upgrade / Uninstall buttons.

## 4. Install the chart

If you pushed to a registry in the previous (optional) section, install from
there: 
a. From portainer WebUI: 
1. Click on your cluster
2. Click on Applications
3. Create from Code > Helm Chart > Select chart museum repo and pick Carbonio

b. Install from registry using CLI
```bash
# from Zot (OCI):
helm install carbonio oci://registry.dev.zextras.com/dev/carbonio \
  --version 0.1.0 -n carbonio
# or from ChartMuseum (or via Portainer's UI):
helm install carbonio chartmuseum/carbonio -n carbonio
```
c. Install from the local `chart/` directory:
```bash
helm install carbonio chart/ -n carbonio
```
Pick groups with the same override file used for podman by adding `-f chart/values.local.yaml`.
Watch pods, then uninstall when done:
```bash
kubectl -n carbonio get pods -w
helm uninstall carbonio -n carbonio
```

## 5. Provisioning

Run `./provision.sh` after the pods are up — it execs `zmprov < zmprov.prov`
inside `carbonio-mailbox`, then refreshes composed-ui's nginx
(`zmproxyconfgen && nginx -s reload`) so the proxy picks up the new server:
```bash
./provision.sh
# override defaults if needed:
NAMESPACE=carbonio PROV_FILE=zmprov.prov ./provision.sh
```
Idempotent — the script probes `gd carbonio.localhost` first and skips zmprov
if the domain already exists, so re-running after an upgrade is safe.

## 6. Access
- Web UI: `https://<vm-ip>` (composed-ui, hostPort 443)
- Grafana: `http://<vm-ip>:3000` · Consul UI: `http://<vm-ip>:8500`

> Folding provisioning into a Helm post-install/post-upgrade Job — and moving
> stateful pods to Deployments/StatefulSets — are the next steps.


