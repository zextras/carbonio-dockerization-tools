#!/bin/sh
TOKEN_FILE="${TOKEN_FILE:-/etc/carbonio/${SERVICE_NAME}/service-discover/token}"

do_setup() {
  echo "[sidecar] Waiting for consul agent..."
  until consul members >/dev/null 2>&1; do sleep 1; done
  echo "[sidecar] Consul agent ready."
  consul services register /consul/config/*.hcl
  if [ -n "${SETUP_SCRIPT}" ]; then
    (${SETUP_SCRIPT} || true)
  fi
}

do_start() {
  echo "[sidecar] Starting envoy for ${SERVICE_NAME}"
  exec consul connect envoy \
    -sidecar-for="${SERVICE_NAME}" \
    -admin-bind=localhost:0 \
    -token-file="$TOKEN_FILE" \
    ${ENVOY_EXTRA_ARGS:-}
}

case "${1:-all}" in
  setup)
    do_setup
    ;;
  start)
    do_start
    ;;
  *)
    do_setup
    do_start
    ;;
esac
