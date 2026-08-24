#!/usr/bin/env bash
# install_k3s.sh — bootstrap a single-node k3s for the Carbonio chart.
# Idempotent: re-running skips steps that already succeeded.
#
# Helm is NOT installed here (its installer is OS-dependent — install it
# yourself per https://helm.sh/docs/intro/install/).
#
# Usage:
#   ./install_k3s.sh
#   REGISTRY_USER=u REGISTRY_PASS=p ./install_k3s.sh
#
# Portainer and ChartMuseum are `make install_portainer` / `make
# install_chartmuseum`, so they work on every platform, not just this script's.
#
# Env overrides:
#   NAMESPACE         (default: carbonio)
#   REGISTRY_SERVER   (default: registry.dev.zextras.com)
#   REGISTRY_SECRET   (default: zextras-registry)
#   REGISTRY_USER     pull-secret username (skipped if empty)
#   REGISTRY_PASS     pull-secret password (skipped if empty)

set -euo pipefail
cd "$(dirname "$0")"

case "${1:-}" in
  "") ;;
  -h|--help) sed -n '2,/^[^#]/ s/^# \{0,1\}//p' "$0"; exit 0 ;;
  *) echo "Unknown argument: $1 (try --help)" >&2; exit 2 ;;
esac

export NAMESPACE="${NAMESPACE:-carbonio}"
export REGISTRY_SERVER="${REGISTRY_SERVER:-registry.dev.zextras.com}"
export REGISTRY_SECRET="${REGISTRY_SECRET:-zextras-registry}"

# 1. k3s
if command -v k3s >/dev/null 2>&1; then
  echo "==> k3s already installed: $(k3s --version | head -1)"
else
  echo "==> Installing k3s (no Traefik, no ServiceLB, kubeconfig 644)..."
  curl -sfL https://get.k3s.io | \
    INSTALL_K3S_EXEC="--disable traefik --disable servicelb --write-kubeconfig-mode=644" sh -
fi

# 2. ~/.kube/config — keep kubectl working without per-shell KUBECONFIG export
if [ -e "$HOME/.kube/config" ] && ! cmp -s "$HOME/.kube/config" /etc/rancher/k3s/k3s.yaml; then
  echo "==> $HOME/.kube/config exists and differs from k3s.yaml — leaving it alone."
  echo "    To target k3s in this shell: export KUBECONFIG=/etc/rancher/k3s/k3s.yaml"
else
  echo "==> Setting $HOME/.kube/config from /etc/rancher/k3s/k3s.yaml..."
  mkdir -p "$HOME/.kube"
  cp /etc/rancher/k3s/k3s.yaml "$HOME/.kube/config"
fi

# helm is required from here on (namespace ops use kubectl, but the optional
# installs and chart use helm; check early so a missing helm fails fast).
if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl not found in PATH — k3s install should have set it up; check your shell." >&2
  exit 1
fi

# 3. Namespace
echo "==> Ensuring namespace $NAMESPACE exists..."
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# 4. Pull secret — same script the Makefile's registry_secret target runs
./registry_secret.sh

echo ""
echo "=== Done ==="
echo "Next: ensure helm is installed, then:"
echo "  make carbonio_install          # or: helm install carbonio chart/ -n $NAMESPACE"
echo "  make install_portainer         # optional web UI"
echo "  make install_chartmuseum       # optional Helm repo (needed to install from Portainer)"
