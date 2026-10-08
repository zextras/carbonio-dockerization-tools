#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

REMOTE_HOST="${REMOTE_HOST:-kc-dev4-u22-ce.demo.zextras.io}"
CONSUL_TOKEN="${CONSUL_TOKEN:?Set CONSUL_TOKEN to the remote Consul ACL token}"
ADVERTISE_ADDR="${ADVERTISE_ADDR:?Set ADVERTISE_ADDR to this machines IP reachable from the remote VM}"
CONSUL_ENCRYPT_KEY="$(cat remote-config/mesh/gossip-key)"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CONSUL_TLS_DIR="$SCRIPT_DIR/remote-config/tls"
CONSUL_CONFIG_DIR="$SCRIPT_DIR/remote-config/rendered"
SERVICE_CONFIG_DIR="$SCRIPT_DIR/remote-config/service-config"
SETUP_SCRIPTS_DIR="$SCRIPT_DIR/templates/scripts"
POD_TEMPLATE=templates/carbonio-catalog-remote.yaml
POD_FILE="$SCRIPT_DIR/remote-config/rendered-pod.yaml"

# Render consul config
mkdir -p "$CONSUL_CONFIG_DIR"
sed \
  -e "s|REMOTE_HOST|${REMOTE_HOST}|g" \
  -e "s|ADVERTISE_ADDR|${ADVERTISE_ADDR}|g" \
  -e "s|CONSUL_TOKEN|${CONSUL_TOKEN}|g" \
  -e "s|CONSUL_ENCRYPT_KEY|${CONSUL_ENCRYPT_KEY}|g" \
  templates/consul-remote.hcl > "$CONSUL_CONFIG_DIR/consul-remote.hcl"

# Render service registration config (for envoy sidecar)
mkdir -p "$SERVICE_CONFIG_DIR"
cat templates/carbonio-catalog-service.hcl > "$SERVICE_CONFIG_DIR/carbonio-catalog-service.hcl"

# Render pod yaml
sed \
  -e "s|CONSUL_TLS_DIR|${CONSUL_TLS_DIR}|g" \
  -e "s|CONSUL_CONFIG_DIR|${CONSUL_CONFIG_DIR}|g" \
  -e "s|SERVICE_CONFIG_DIR|${SERVICE_CONFIG_DIR}|g" \
  -e "s|SETUP_SCRIPTS_DIR|${SETUP_SCRIPTS_DIR}|g" \
  -e "s|CONSUL_TOKEN|${CONSUL_TOKEN}|g" \
  "$POD_TEMPLATE" > "$POD_FILE"

echo "==> Remote host:      $REMOTE_HOST"
echo "==> Advertise address: $ADVERTISE_ADDR"
echo "==> Consul token:      ${CONSUL_TOKEN:0:8}..."
echo "==> TLS certs:         $CONSUL_TLS_DIR"
echo "==> Consul config:     $CONSUL_CONFIG_DIR"

case "${1:-help}" in
  up)
    echo "==> Starting carbonio-catalog with host networking..."
    podman play kube "$POD_FILE" --network host
    echo ""
    echo "==> Waiting for consul agent to join remote server..."
    for i in $(seq 1 30); do
      if podman exec carbonio-catalog-consul-agent consul members 2>/dev/null | grep -q "$REMOTE_HOST\|alive"; then
        echo "    Consul agent joined successfully."
        break
      fi
      if [ "$i" -eq 30 ]; then
        echo "    WARNING: could not confirm consul join (check logs)"
      fi
      sleep 2
    done

    echo ""
    echo "=== carbonio-catalog running (remote consul) ==="
    echo "  Remote consul: $REMOTE_HOST"
    echo "  Logs:          $0 logs [container]"
    echo "  Down:          $0 down"
    ;;

  down)
    echo "==> Tearing down..."
    podman play kube "$POD_FILE" --down 2>/dev/null || true
    rm -f "$POD_FILE"
    echo "Done."
    ;;

  status)
    echo "==> Pod:"
    podman pod inspect carbonio-catalog 2>/dev/null | jq '{Name: .Name, State: .State}' || echo "    not running"
    echo ""
    echo "==> Containers:"
    podman ps -a --pod --filter pod=carbonio-catalog --format "table {{.PodName}}\t{{.Names}}\t{{.Status}}"
    echo ""
    echo "==> Consul members:"
    podman exec carbonio-catalog-consul-agent consul members 2>/dev/null || echo "    consul agent not running"
    ;;

  logs)
    CONTAINER="${2:-}"
    if [ -n "$CONTAINER" ]; then
      podman logs "carbonio-catalog-${CONTAINER}"
    else
      for c in $(podman pod inspect carbonio-catalog 2>/dev/null | jq -r '.Containers[].Name' 2>/dev/null); do
        echo "--- $c ---"
        podman logs --tail 20 "$c" 2>/dev/null
        echo ""
      done
    fi
    ;;

  *)
    echo "Usage: $0 {up|down|status|logs}"
    echo ""
    echo "Required env vars:"
    echo "  CONSUL_TOKEN     - ACL token from the remote Consul server"
    echo "  ADVERTISE_ADDR   - This machines IP reachable from the remote VM"
    echo ""
    echo "Optional:"
    echo "  REMOTE_HOST      - Remote Consul server (default: kc-dev4-u22-ce.demo.zextras.io)"
    echo ""
    echo "Example:"
    echo "  CONSUL_TOKEN=abc123 ADVERTISE_ADDR=192.168.1.50 $0 up"
    ;;
esac
