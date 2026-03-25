#!/bin/sh
echo '[sidecar] Waiting for consul agent...'
until consul members >/dev/null 2>&1; do sleep 1; done
echo '[sidecar] Consul agent ready.'
consul services register /consul/config/*.hcl
if [ -n "${SETUP_SCRIPT}" ]; then
  (${SETUP_SCRIPT} || true)
fi
echo "[sidecar] Starting envoy for ${SERVICE_NAME}"
exec consul connect envoy \
  -sidecar-for="${SERVICE_NAME}" \
  -admin-bind=localhost:0 \
  -token-file="${TOKEN_FILE:-/etc/carbonio/${SERVICE_NAME}/service-discover/token}" \
  ${ENVOY_EXTRA_ARGS:-}
