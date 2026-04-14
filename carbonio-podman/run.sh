#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

NETWORK="carbonio-mesh"
MAILBOX_REPO="${MAILBOX_REPO:-../../carbonio-mailbox}"
COMPOSED_UI_DIR="${COMPOSED_UI_DIR:-../composed-ui}"
DOCS_EDITOR_DIR="${DOCS_EDITOR_DIR:-../carbonio-docs-editor}"
TAG="${CARBONIO_GLOBAL_IMAGES_TAG:-devel}"
REGISTRY="registry.dev.zextras.com/dev"

# Pod groups in startup order (each directory is played in full)
POD_GROUPS=(
  pods/common
  pods/mails
  pods/files
  pods/wsc
  pods/ui
)

play_group() {
  local dir="$1" action="${2:-up}"
  for f in "$dir"/*.yaml; do
    [ -f "$f" ] || continue
    if [ "$action" = "up" ]; then
      echo "    Starting $(basename "$f" .yaml)..."
      podman play kube "$f" --network "$NETWORK" --configmap configmaps/consul-acl.yaml -q >/dev/null || echo "    WARNING: $f failed to start"
    else
      podman play kube "$f" --down ${3:-} 2>/dev/null || true
    fi
  done
}

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

    echo "==> Building carbonio-docs-editor image..."
    podman build \
      -t localhost/carbonio-docs-editor:latest \
      -f "$DOCS_EDITOR_DIR/Dockerfile" \
      "$DOCS_EDITOR_DIR"

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

    echo "==> Starting pod groups..."
    for group in "${POD_GROUPS[@]}"; do
      echo "  [$group]"
      play_group "$group" up
      if [ "$group" = "pods/common" ]; then
        wait_consul
      fi
    done

    echo "==> Waiting for mailbox to be ready..."
    for i in $(seq 1 120); do
      result=$(echo 'gd carbonio.localhost' | podman exec -i carbonio-mailbox-app zmprov 2>&1) || true
      if echo "$result" | grep -q 'NO_SUCH_DOMAIN'; then
        echo "==> Provisioning..."
        podman exec carbonio-mailbox-app sh -c "zmprov < /tmp/provisioning.txt"
        break
      elif echo "$result" | grep -qi 'carbonio.localhost'; then
        echo "    Mailbox ready (domain already provisioned)."
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
    FORCE=""
    if [ "${2:-}" = "--force" ]; then
      FORCE="--force"
      echo "==> Tearing down (removing volumes)..."
    else
      echo "==> Tearing down (preserving volumes)..."
    fi
    for (( i=${#POD_GROUPS[@]}-1; i>=0; i-- )); do
      play_group "${POD_GROUPS[$i]}" down $FORCE
    done
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

  restart)
    FILE="${2:-}"
    if [ -z "$FILE" ]; then
      echo "Usage: $0 restart <yaml-file>"
      echo "  e.g.: $0 restart pods/wsc/carbonio-message-dispatcher.yaml"
      exit 1
    fi
    if [ ! -f "$FILE" ]; then
      echo "File not found: $FILE"
      exit 1
    fi
    echo "==> Restarting $(basename "$FILE" .yaml)..."
    podman play kube "$FILE" --down 2>/dev/null || true
    podman play kube "$FILE" --network "$NETWORK" --configmap configmaps/consul-acl.yaml
    echo "Done."
    ;;

  *)
    echo "Usage: $0 {up|down|status|logs|restart}"
    ;;
esac
