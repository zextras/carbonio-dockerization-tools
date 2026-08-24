#!/usr/bin/env bash
# Create the private-registry pull secret, if credentials are available.
# Single implementation shared by install_k3s.sh and the Makefile's
# registry_secret target (which on macOS feeds it the keychain credential).
# No-op with a hint when REGISTRY_USER/REGISTRY_PASS are unset; idempotent.
#
# Env: NAMESPACE, REGISTRY_SERVER, REGISTRY_SECRET, REGISTRY_USER, REGISTRY_PASS

set -euo pipefail

NAMESPACE="${NAMESPACE:-carbonio}"
REGISTRY_SERVER="${REGISTRY_SERVER:-registry.dev.zextras.com}"
REGISTRY_SECRET="${REGISTRY_SECRET:-zextras-registry}"

if [ -z "${REGISTRY_USER:-}" ] || [ -z "${REGISTRY_PASS:-}" ]; then
  echo "==> No registry credentials — skipping pull secret."
  echo "    Set REGISTRY_USER + REGISTRY_PASS (on macOS an existing docker login is read from the keychain)."
  echo "    Create it before installing the chart, or re-run this with them set."
  exit 0
fi

echo "==> Creating $REGISTRY_SECRET in $NAMESPACE for $REGISTRY_USER..."
kubectl -n "$NAMESPACE" create secret docker-registry "$REGISTRY_SECRET" \
  --docker-server="$REGISTRY_SERVER" \
  --docker-username="$REGISTRY_USER" \
  --docker-password="$REGISTRY_PASS" \
  --dry-run=client -o yaml | kubectl apply -f -
