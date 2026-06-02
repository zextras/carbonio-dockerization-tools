#!/usr/bin/env bash
# Provision Carbonio on k3s: run zmprov.prov inside carbonio-mailbox, then
# refresh composed-ui's nginx so the proxy picks up the new server.
# Idempotent — probes `gd carbonio.localhost` first; skips zmprov if the
# domain already exists. (start.sh handles the same flow for podman.)
#
# Usage:
#   ./provision.sh
#   NAMESPACE=carbonio PROV_FILE=chart/files/zmprov.prov ./provision.sh
set -euo pipefail
cd "$(dirname "$0")"

NAMESPACE="${NAMESPACE:-carbonio}"
PROV_FILE="${PROV_FILE:-zmprov.prov}"

echo "==> Waiting for carbonio-mailbox to be Ready..."
kubectl -n "$NAMESPACE" wait --for=condition=Ready pod/carbonio-mailbox --timeout=600s

echo "==> Probing zmprov readiness..."
PROVISIONED=0
for i in $(seq 1 120); do
  result=$(kubectl -n "$NAMESPACE" exec -i carbonio-mailbox -c app -- \
    sh -c 'echo "gd carbonio.localhost" | zmprov 2>&1') || true
  if echo "$result" | grep -q 'NO_SUCH_DOMAIN'; then
    echo "==> Running zmprov.prov..."
    kubectl -n "$NAMESPACE" exec -i carbonio-mailbox -c app -- zmprov < "$PROV_FILE"
    PROVISIONED=1
    break
  elif echo "$result" | grep -qi 'carbonio.localhost'; then
    echo "==> Mailbox already provisioned, skipping zmprov."
    break
  fi
  sleep 2
done

if [ "$PROVISIONED" -eq 1 ]; then
  echo "==> Waiting for carbonio-composed-ui to be Ready..."
  kubectl -n "$NAMESPACE" wait --for=condition=Ready pod/carbonio-composed-ui --timeout=300s

  echo "==> Regenerating nginx config + reloading composed-ui..."
  kubectl -n "$NAMESPACE" exec carbonio-composed-ui -c app -- sh -c \
    '/usr/bin/zmproxyconfgen && /opt/zextras/common/sbin/nginx -c /opt/zextras/conf/nginx.conf -s reload'
fi

echo "==> Done."
