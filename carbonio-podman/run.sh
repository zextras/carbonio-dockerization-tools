#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

NETWORK="carbonio-mesh"
MAILBOX_REPO="${MAILBOX_REPO:-../../carbonio-mailbox}"
COMPOSED_UI_DIR="${COMPOSED_UI_DIR:-../composed-ui}"
TAG="${CARBONIO_GLOBAL_IMAGES_TAG:-devel}"
REGISTRY="registry.dev.zextras.com/dev"

# Consul server must start first
CONSUL_POD=pods/consul-server.yaml

# Remaining pods in startup order
PODS=(
  pods/carbonio-monitoring.yaml
  pods/carbonio-catalog.yaml
  pods/carbonio-postgres.yaml
  pods/carbonio-openldap.yaml
  pods/carbonio-mariadb.yaml
  pods/carbonio-postfix.yaml
  pods/carbonio-mailbox.yaml
  pods/carbonio-user-management.yaml
  pods/carbonio-storages.yaml
  pods/carbonio-files.yaml
  pods/carbonio-composed-ui.yaml
)

wait_consul() {
  echo "==> Waiting for consul-server..."
  for i in $(seq 1 60); do
    if curl -sf http://localhost:8500/v1/status/leader 2>/dev/null | grep -q ':'; then
      echo "    Consul server ready."
      return 0
    fi
    sleep 1
  done
  echo "    ERROR: consul-server did not become ready"
  return 1
}

case "${1:-help}" in
  up)
    echo "==> Creating network..."
    podman network create "$NETWORK" 2>/dev/null || true

    echo "==> Building composed-ui image..."
    podman build \
      -t localhost/carbonio-composed-ui:latest \
      -f "$COMPOSED_UI_DIR/Dockerfile" \
      --build-arg CARBONIO_SHELL_UI_IMAGE="${CARBONIO_SHELL_UI_IMAGE:-$REGISTRY/carbonio-shell-ui:$TAG}" \
      --build-arg CARBONIO_MAILS_UI_IMAGE="${CARBONIO_MAILS_UI_IMAGE:-$REGISTRY/carbonio-mails-ui:$TAG}" \
      --build-arg CARBONIO_CONTACTS_UI_IMAGE="${CARBONIO_CONTACTS_UI_IMAGE:-$REGISTRY/carbonio-contacts-ui:$TAG}" \
      --build-arg CARBONIO_CALENDARS_UI_IMAGE="${CARBONIO_CALENDARS_UI_IMAGE:-$REGISTRY/carbonio-calendars-ui:$TAG}" \
      --build-arg CARBONIO_AUTH_UI_IMAGE="${CARBONIO_AUTH_UI_IMAGE:-$REGISTRY/carbonio-auth-ui:$TAG}" \
      --build-arg CARBONIO_SEARCH_UI_IMAGE="${CARBONIO_SEARCH_UI_IMAGE:-$REGISTRY/carbonio-search-ui:$TAG}" \
      --build-arg CARBONIO_FILES_UI_IMAGE="${CARBONIO_FILES_UI_IMAGE:-$REGISTRY/carbonio-files-ui:$TAG}" \
      --build-arg CARBONIO_TASKS_UI_IMAGE="${CARBONIO_TASKS_UI_IMAGE:-$REGISTRY/carbonio-tasks-ui:$TAG}" \
      --build-arg CARBONIO_WSC_UI_IMAGE="${CARBONIO_WSC_UI_IMAGE:-$REGISTRY/carbonio-ws-collaboration-ui:$TAG}" \
      --build-arg CARBONIO_STORAGES_UI_IMAGE="${CARBONIO_STORAGES_UI_IMAGE:-$REGISTRY/carbonio-storages-ui:$TAG}" \
      --build-arg CARBONIO_LOGIN_UI_IMAGE="${CARBONIO_LOGIN_UI_IMAGE:-$REGISTRY/carbonio-login-ui:$TAG}" \
      --build-arg CARBONIO_ADMIN_CONSOLE_UI_IMAGE="${CARBONIO_ADMIN_CONSOLE_UI_IMAGE:-$REGISTRY/carbonio-admin-console-ui:$TAG}" \
      --build-arg CARBONIO_ADMIN_LOGIN_UI_IMAGE="${CARBONIO_ADMIN_LOGIN_UI_IMAGE:-$REGISTRY/carbonio-admin-login-ui:$TAG}" \
      --build-arg CARBONIO_WEBUI_I18N_IMAGE="${CARBONIO_WEBUI_I18N_IMAGE:-$REGISTRY/carbonio-webui-i18n:$TAG}" \
      --build-arg CARBONIO_PROXY_IMAGE="${CARBONIO_PROXY_IMAGE:-$REGISTRY/carbonio-proxy:$TAG}" \
      "$COMPOSED_UI_DIR"

    echo "==> Starting consul-server..."
    podman play kube "$CONSUL_POD" --network "$NETWORK"
    wait_consul
    echo "==> Initializing Consul KV..."
    bash consul-kv/init.sh || echo "    WARNING: KV init had errors (non-fatal)"

    echo "==> Starting pods..."
    for pod in "${PODS[@]}"; do
      echo "    Starting $(basename "$pod" .yaml)..."
      podman play kube "$pod" --network "$NETWORK" || echo "    WARNING: $pod failed to start"
    done

    echo "==> Waiting for mailbox to be ready..."
    for i in $(seq 1 120); do
      if echo 'gd carbonio.localhost' | podman exec -i carbonio-mailbox-app zmprov 2>&1 | grep -q 'NO_SUCH_DOMAIN'; then
        echo "==> Provisioning..."
        podman exec carbonio-mailbox-app sh -c "zmprov < /tmp/provisioning.txt"
        break
      fi
      if [ "$i" -eq 120 ]; then
        echo "    ERROR: mailbox did not become ready"
        exit 1
      fi
      sleep 2
    done

    echo ""
    echo "=== Mailbox POC running ==="
    echo "  Consul UI:     http://localhost:8500"
    echo "  Grafana:       http://localhost:3000  (admin/admin)"
    echo "  Mailbox:       http://localhost:8080"
    echo "  Web UI:        https://localhost:443"
    echo ""
    echo "  Status:  $0 status"
    echo "  Logs:    $0 logs <pod> [container]"
    echo "  Down:    $0 down"
    ;;

  down)
    echo "==> Tearing down..."
    for (( i=${#PODS[@]}-1; i>=0; i-- )); do
      podman play kube "${PODS[$i]}" --down 2>/dev/null || true
    done
    podman play kube "$CONSUL_POD" --down 2>/dev/null || true
    podman network rm "$NETWORK" 2>/dev/null || true
    echo "Done."
    ;;

  status)
    echo "==> Pods:"
    podman pod ps
    echo ""
    echo "==> Containers:"
    podman ps -a --pod --format "table {{.PodName}}\t{{.Names}}\t{{.Status}}"
    echo ""
    echo "==> Consul members:"
    podman exec consul-server-consul consul members 2>/dev/null || echo "    consul-server not running"
    echo ""
    echo "==> Registered services:"
    curl -s http://localhost:8500/v1/catalog/services 2>/dev/null | python3 -m json.tool 2>/dev/null || echo "    consul not reachable"
    ;;

  logs)
    POD="${2:-}"
    CONTAINER="${3:-}"
    if [ -z "$POD" ]; then
      echo "Usage: $0 logs <pod-name> [container-name]"
      echo "  e.g.: $0 logs carbonio-mailbox sidecar"
      exit 1
    fi
    if [ -n "$CONTAINER" ]; then
      podman logs "${POD}-${CONTAINER}"
    else
      for c in $(podman pod inspect "$POD" 2>/dev/null | jq -r '.Containers[].Name' 2>/dev/null); do
        echo "--- $c ---"
        podman logs --tail 20 "$c" 2>/dev/null
        echo ""
      done
    fi
    ;;

  *)
    echo "Usage: $0 {up|down|status|logs}"
    ;;
esac
