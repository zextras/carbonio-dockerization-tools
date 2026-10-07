#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

REMOTE_HOST="${REMOTE_HOST:-kc-dev4-u22-ce.demo.zextras.io}"
REMOTE_USER="${REMOTE_USER:-root}"
MESH_PASSWORD="${MESH_PASSWORD:?Set MESH_PASSWORD to the service-discover GPG passphrase}"

MESH_DIR="remote-config/mesh"
TLS_DIR="remote-config/tls"

echo "==> Fetching LDAP password from ${REMOTE_HOST}..."
LDAP_PASSWORD=$(ssh "${REMOTE_USER}@${REMOTE_HOST}" "/opt/zextras/bin/zmlocalconfig -s -m nokey ldap_root_password")
echo "    Got LDAP password."

echo "==> Fetching mesh credentials from LDAP (via container)..."
mkdir -p remote-config
podman run --rm --entrypoint ldapsearch -v "$(pwd)/remote-config:/out:z" \
  docker.io/osixia/openldap:latest \
  -x -H "ldap://${REMOTE_HOST}:389" \
    -D "cn=config" -w "${LDAP_PASSWORD}" \
    -b "cn=config,cn=zimbra" -s base \
    -LLL -tt -T /out carbonioMeshCredentials > /dev/null

echo "==> Decrypting and extracting mesh credentials..."
mkdir -p "$MESH_DIR"
base64 -d remote-config/ldapsearch-carbonioMeshCredentials-* \
  | gpg --batch --passphrase "${MESH_PASSWORD}" -d 2>/dev/null \
  | tar xf - -C "$MESH_DIR"
rm remote-config/ldapsearch-carbonioMeshCredentials-*

echo "==> Generating client TLS certificate..."
mkdir -p "$TLS_DIR"
CA_CERT="$MESH_DIR/var/lib/service-discover/consul-agent-ca.pem"
CA_KEY="$MESH_DIR/var/lib/service-discover/consul-agent-ca-key.pem"

openssl ecparam -name prime256v1 -genkey -noout -out "$TLS_DIR/client-key.pem"
openssl req -new -key "$TLS_DIR/client-key.pem" -out "$TLS_DIR/client.csr" -subj "/CN=client.dc1.consul"
openssl x509 -req -in "$TLS_DIR/client.csr" \
  -CA "$CA_CERT" -CAkey "$CA_KEY" \
  -CAcreateserial -out "$TLS_DIR/client.pem" -days 365 -sha256 2>/dev/null
rm "$TLS_DIR/client.csr"

cp "$CA_CERT" "$CA_KEY" "$TLS_DIR/"
chmod 644 "$TLS_DIR"/*.pem

# Get consul bootstrap token from the remote VM
echo "==> Fetching consul bootstrap token..."
CONSUL_TOKEN=$(ssh "${REMOTE_USER}@${REMOTE_HOST}" "service-discover bootstrap-token --password='${MESH_PASSWORD}'")

echo ""
echo "=== Setup complete ==="
echo "  Mesh config:  $MESH_DIR"
echo "  TLS certs:    $TLS_DIR"
echo "  Consul token: $CONSUL_TOKEN"
echo ""
echo "Now run:"
echo "  CONSUL_TOKEN=$CONSUL_TOKEN ADVERTISE_ADDR=<your-vpn-ip> ./run.sh up"
