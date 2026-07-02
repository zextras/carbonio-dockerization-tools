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
#   ./install_k3s.sh --with-portainer --with-chartmuseum
#
# Env overrides:
#   NAMESPACE         (default: carbonio)
#   REGISTRY_SERVER   (default: registry.dev.zextras.com)
#   REGISTRY_SECRET   (default: zextras-registry)
#   REGISTRY_USER     pull-secret username (skipped if empty)
#   REGISTRY_PASS     pull-secret password (skipped if empty)

set -euo pipefail
cd "$(dirname "$0")"

WITH_PORTAINER=0
WITH_CHARTMUSEUM=0
for arg in "$@"; do
  case "$arg" in
    --with-portainer)   WITH_PORTAINER=1 ;;
    --with-chartmuseum) WITH_CHARTMUSEUM=1 ;;
    -h|--help)          sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Unknown flag: $arg (try --help)" >&2; exit 2 ;;
  esac
done

NAMESPACE="${NAMESPACE:-carbonio}"
REGISTRY_SERVER="${REGISTRY_SERVER:-registry.dev.zextras.com}"
REGISTRY_SECRET="${REGISTRY_SECRET:-zextras-registry}"

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

# 4. Pull secret (only if creds were provided)
if [ -n "${REGISTRY_USER:-}" ] && [ -n "${REGISTRY_PASS:-}" ]; then
  echo "==> Creating docker-registry secret $REGISTRY_SECRET in $NAMESPACE..."
  kubectl -n "$NAMESPACE" create secret docker-registry "$REGISTRY_SECRET" \
    --docker-server="$REGISTRY_SERVER" \
    --docker-username="$REGISTRY_USER" \
    --docker-password="$REGISTRY_PASS" \
    --dry-run=client -o yaml | kubectl apply -f -
else
  echo "==> REGISTRY_USER/REGISTRY_PASS not set — skipping pull secret."
  echo "    Re-run with them set, or create the secret manually before installing the chart."
fi

# 5. Optional: Portainer
if [ "$WITH_PORTAINER" = "1" ]; then
  command -v helm >/dev/null 2>&1 || {
    echo "helm not found — install it first (https://helm.sh/docs/intro/install/)" >&2
    exit 1
  }
  echo "==> Installing Portainer (NodePort)..."
  helm repo add portainer https://portainer.github.io/k8s/ 2>/dev/null || true
  helm repo update portainer
  helm upgrade --install portainer portainer/portainer \
    -n portainer --create-namespace \
    --set service.type=NodePort
fi

# 6. Optional: ChartMuseum (NodePort 30808)
if [ "$WITH_CHARTMUSEUM" = "1" ]; then
  command -v helm >/dev/null 2>&1 || {
    echo "helm not found — install it first (https://helm.sh/docs/intro/install/)" >&2
    exit 1
  }
  echo "==> Installing ChartMuseum (NodePort 30808)..."
  helm repo add chartmuseum https://chartmuseum.github.io/charts 2>/dev/null || true
  helm repo update chartmuseum
  helm upgrade --install chartmuseum chartmuseum/chartmuseum \
    -n chartmuseum --create-namespace \
    --set env.open.DISABLE_API=false \
    --set env.open.ALLOW_OVERWRITE=true \
    --set persistence.enabled=true \
    --set service.type=NodePort \
    --set service.nodePort=30808
fi

echo ""
echo "=== Done ==="
echo "Next: ensure helm is installed, then:"
echo "  helm install carbonio chart/ -n $NAMESPACE"
