#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

NETWORK="carbonio-mesh"
MAILBOX_REPO="${MAILBOX_REPO:-../../carbonio-mailbox}"
COMPOSED_UI_DIR="${COMPOSED_UI_DIR:-../composed-ui}"
DOCS_EDITOR_DIR="${DOCS_EDITOR_DIR:-../carbonio-docs-editor}"
TAG="${CARBONIO_GLOBAL_IMAGES_TAG:-devel}"
REGISTRY="registry.dev.zextras.com/dev"
PULL_POLICY="always"

# Sets SELECTED_APP_GROUPS from a list of group names, deduplicated and in
# canonical startup order. `all` expands to mails+files+wsc.
SELECTED_APP_GROUPS=()
resolve_app_groups() {
  local want_mails=0 want_files=0 want_wsc=0
  local g
  for g in "$@"; do
    case "$g" in
      all)   want_mails=1; want_files=1; want_wsc=1 ;;
      mails) want_mails=1 ;;
      files) want_files=1 ;;
      wsc)   want_wsc=1 ;;
      *) echo "Unknown group: $g (expected: all|mails|files|wsc)" >&2; return 1 ;;
    esac
  done
  SELECTED_APP_GROUPS=()
  [ $want_mails -eq 1 ] && SELECTED_APP_GROUPS+=(pods/mails)
  [ $want_files -eq 1 ] && SELECTED_APP_GROUPS+=(pods/files)
  [ $want_wsc   -eq 1 ] && SELECTED_APP_GROUPS+=(pods/wsc)
  return 0
}

contains() {
  local needle="$1"; shift
  local x
  for x in "$@"; do [ "$x" = "$needle" ] && return 0; done
  return 1
}

play_group() {
  local dir="$1" action="${2:-up}"
  for f in "$dir"/*.yaml; do
    [ -f "$f" ] || continue
    if [ "$action" = "up" ]; then
      echo "    Starting $(basename "$f" .yaml)..."
      if [ "$PULL_POLICY" = "missing" ]; then
        sed 's/imagePullPolicy: Always/imagePullPolicy: IfNotPresent/g' "$f" \
          | podman play kube - --network "$NETWORK" --configmap configmaps/consul-acl.yaml -q >/dev/null \
          || echo "    WARNING: $f failed to start"
      else
        podman play kube "$f" --network "$NETWORK" --configmap configmaps/consul-acl.yaml -q >/dev/null \
          || echo "    WARNING: $f failed to start"
      fi
    else
      podman play kube "$f" --down ${3:-} 2>/dev/null || true
    fi
  done
}

play_one() {
  local f="$1"
  echo "    Starting $(basename "$f" .yaml)..."
  if [ "$PULL_POLICY" = "missing" ]; then
    sed 's/imagePullPolicy: Always/imagePullPolicy: IfNotPresent/g' "$f" \
      | podman play kube - --network "$NETWORK" --configmap configmaps/consul-acl.yaml -q >/dev/null \
      || echo "    WARNING: $f failed to start"
  else
    podman play kube "$f" --network "$NETWORK" --configmap configmaps/consul-acl.yaml -q >/dev/null \
      || echo "    WARNING: $f failed to start"
  fi
}

play_mails() {
  play_one pods/mails/carbonio-mailbox.yaml
  play_one pods/mails/carbonio-postfix.yaml

  wait_mailbox_ready_and_provision || return 1

  local expected_servers=(carbonio-mailbox)
  if [ "${MULTI:-0}" -eq 1 ]; then
    play_one pods/mails/carbonio-mailbox-2.yaml
    expected_servers+=(carbonio-mailbox-2)
  fi
  wait_mailbox_servers "${expected_servers[@]}" || return 1
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

wait_mailbox_ready_and_provision() {
  echo "==> Waiting for mailbox to be ready..."
  for i in $(seq 1 120); do
    result=$(echo 'gd carbonio.localhost' | podman exec -i carbonio-mailbox-app zmprov 2>&1) || true
    if echo "$result" | grep -q 'NO_SUCH_DOMAIN'; then
      echo "==> Provisioning..."
      printf '%s' "$(cat zmprov.prov)" | podman exec -i carbonio-mailbox-app zmprov
      return 0
    elif echo "$result" | grep -qi 'carbonio.localhost'; then
      echo "    Mailbox ready (domain already provisioned)."
      return 0
    fi
    if [ "$i" -eq 120 ]; then
      echo "    ERROR: mailbox did not become ready"
      return 1
    fi
    sleep 2
  done
}

wait_mailbox_servers() {
  local expected=("$@")
  echo "==> Waiting for mailbox servers in LDAP: ${expected[*]}"
  for i in $(seq 1 60); do
    result=$(podman exec -i carbonio-mailbox-app zmprov -l gas 2>/dev/null) || result=""
    local missing=0
    for s in "${expected[@]}"; do
      if ! echo "$result" | grep -qx "$s"; then
        missing=1
        break
      fi
    done
    if [ $missing -eq 0 ]; then
      echo "    All mailbox servers registered."
      return 0
    fi
    sleep 2
  done
  echo "    ERROR: not all expected mailbox servers registered in time"
  echo "    zmprov gas returned: $result"
  return 1
}


case "${1:-help}" in
  up)
    UP_ARGS=()
    MULTI=0
    for arg in "${@:2}"; do
      case "$arg" in
        --multi) MULTI=1 ;;
        --missing) PULL_POLICY="missing" ;;
        *) UP_ARGS+=("$arg") ;;
      esac
    done
    export MULTI
    [ ${#UP_ARGS[@]} -eq 0 ] && UP_ARGS=(all)
    resolve_app_groups "${UP_ARGS[@]}" || exit 1
    SELECTED_GROUPS=(pods/common "${SELECTED_APP_GROUPS[@]}" pods/ui)

    echo "==> Selected groups: ${SELECTED_GROUPS[*]}"
    echo "==> Creating network..."
    podman network create "$NETWORK" 2>/dev/null || true

    echo "==> Building carbonio-docs-editor image..."
    podman build \
      -t localhost/carbonio-docs-editor:latest \
      -f "$DOCS_EDITOR_DIR/Dockerfile" \
      "$DOCS_EDITOR_DIR"

    echo "==> Building composed-ui image (pull=$PULL_POLICY)..."
    podman build \
      -t localhost/carbonio-composed-ui:latest \
      -f "$COMPOSED_UI_DIR/Dockerfile" \
      --pull=$PULL_POLICY \
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
    for group in "${SELECTED_GROUPS[@]}"; do
      echo "  [$group]"
      if [ "$group" = "pods/mails" ]; then
        play_mails || exit 1
      else
        play_group "$group" up
      fi
      if [ "$group" = "pods/common" ]; then
        wait_consul
      fi
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
    DOWN_GROUPS=()
    FULL_TEARDOWN=0
    for arg in "${@:2}"; do
      case "$arg" in
        --force) FORCE="--force" ;;
        all) FULL_TEARDOWN=1; DOWN_GROUPS+=(all) ;;
        mails|files|wsc) DOWN_GROUPS+=("$arg") ;;
        *) echo "Unknown arg: $arg (expected: all|mails|files|wsc|--force)" >&2; exit 1 ;;
      esac
    done
    if [ ${#DOWN_GROUPS[@]} -eq 0 ]; then
      FULL_TEARDOWN=1
      DOWN_GROUPS=(all)
    fi
    resolve_app_groups "${DOWN_GROUPS[@]}" || exit 1

    if [ $FULL_TEARDOWN -eq 1 ]; then
      SELECTED_GROUPS=(pods/common "${SELECTED_APP_GROUPS[@]}" pods/ui)
    else
      SELECTED_GROUPS=("${SELECTED_APP_GROUPS[@]}")
    fi

    if [ -n "$FORCE" ]; then
      echo "==> Tearing down ${SELECTED_GROUPS[*]} (removing volumes)..."
    else
      echo "==> Tearing down ${SELECTED_GROUPS[*]} (preserving volumes)..."
    fi
    for (( i=${#SELECTED_GROUPS[@]}-1; i>=0; i-- )); do
      play_group "${SELECTED_GROUPS[$i]}" down $FORCE
    done
    if [ $FULL_TEARDOWN -eq 1 ]; then
      podman network rm "$NETWORK" 2>/dev/null || true
    fi
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

  stats)
    podman stats --no-stream --format "{{.Name}}: {{.MemUsage}}"
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
    echo "Usage: $0 {up [groups...] [--multi] [--missing] | down [groups...] [--force] | status | stats | logs <pod> [container] | restart <yaml>}"
    echo "  groups:    all | mails | files | wsc   (multiple allowed; defaults to 'all' when omitted)"
    echo "  --multi:   also start carbonio-mailbox-2 (mails group only)"
    echo "  --missing: only pull images that are not present locally (default: always pull)"
    echo "  e.g.:   $0 up                # same as: $0 up all"
    echo "          $0 up mails files"
    echo "          $0 up mails --multi"
    echo "          $0 up --missing"
    echo "          $0 down wsc"
    echo "          $0 down              # same as: $0 down all"
    echo "          $0 down --force"
    ;;
esac
