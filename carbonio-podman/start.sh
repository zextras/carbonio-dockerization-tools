#!/usr/bin/env bash
# Start Carbonio on podman straight from the Helm chart (chart/ is the source of
# truth). The only thing forced here is platform=podman (the chart defaults to
# k3s); which groups start, the UI image, etc. all come from chart/values.yaml.
#
# Prereqs: helm + podman; `podman login registry.dev.zextras.com`; and
# carbonio-composed-ui:devel pushed (or override ui.image in values.yaml).
#
# Usage:
#   ./start.sh          # bring up the groups values.yaml enables
#   ./start.sh down     # tear it down
set -euo pipefail
cd "$(dirname "$0")"

NETWORK="carbonio-mesh"

# Optional override file, deep-merged over chart/values.yaml (only the fields you
# set there are overridden). Defaults to chart/values.local.yaml when present;
# point elsewhere with VALUES_FILE=/path/to/file.
VALUES_FILE="${VALUES_FILE:-chart/values.local.yaml}"
render() {
  if [ -f "$VALUES_FILE" ]; then
    helm template carbonio chart/ --set platform=podman -f "$VALUES_FILE"
  else
    helm template carbonio chart/ --set platform=podman
  fi
}

case "${1:-}" in
  -h|--help|help)
    echo "Usage: ./start.sh [up|down] [--force]"
    echo "  up              render chart/ for podman and start the pods (default)"
    echo "  down [--force]  stop the pods (--force also removes volumes)"
    echo ""
    echo "Groups/images come from chart/values.yaml; override with chart/values.local.yaml"
    echo "or VALUES_FILE=/path/to/values.yaml ./start.sh"
    exit 0 ;;
esac

if [ "${1:-up}" = "down" ]; then
  # podman keeps volumes on --down (like k8s keeps PVCs); --force wipes them too.
  FORCE=""
  [ "${2:-}" = "--force" ] && FORCE="--force"
  echo "==> Tearing down${FORCE:+ (removing volumes)}..."
  render | podman play kube - --down $FORCE || true
  echo "Done."
  exit 0
fi

echo "==> Creating network ${NETWORK}..."
podman network create "$NETWORK" 2>/dev/null || true

echo "==> Rendering chart (platform=podman, groups from values.yaml) and starting pods..."
render | podman play kube - --network "$NETWORK"

# Provision the mailbox if this deployment includes it (same step run.sh does).
PROVISIONED=0
if podman container exists carbonio-mailbox-app; then
  echo "==> Waiting for mailbox to be ready..."
  for i in $(seq 1 120); do
    result=$(echo 'gd carbonio.localhost' | podman exec -i carbonio-mailbox-app zmprov 2>&1) || true
    if echo "$result" | grep -q 'NO_SUCH_DOMAIN'; then
      echo "==> Provisioning domain + accounts..."
      printf '%s' "$(cat zmprov.prov)" | podman exec -i carbonio-mailbox-app zmprov
      PROVISIONED=1
      break
    elif echo "$result" | grep -qi 'carbonio.localhost'; then
      echo "    Mailbox already provisioned."
      break
    fi
    if [ "$i" -eq 120 ]; then
      echo "    ERROR: mailbox did not become ready in time"
      exit 1
    fi
    sleep 2
  done
fi

# composed-ui (proxy) renders its nginx config from LDAP at startup. In the
# one-shot play it boots before provisioning, so restart it once afterwards to
# regenerate against the now-provisioned domain/mailbox-server.
if [ "$PROVISIONED" -eq 1 ] && podman pod exists carbonio-composed-ui; then
  echo "==> Restarting composed-ui to pick up the provisioned config..."
  podman pod restart carbonio-composed-ui >/dev/null 2>&1 || true
fi

echo ""
echo "=== Carbonio (from chart) running ==="
echo "  Web UI:   https://localhost"
echo "  Consul:   http://localhost:8500"
echo "  Grafana:  http://localhost:3000  (admin/admin)"
echo "  Teardown: $0 down"
