
CONSUL_HTTP_ADDR="http://localhost:8500"
curl -X PUT --data "carbonio-docs-connector-db" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-docs-connector/database/credentials/db-name"
curl -X PUT --data "admin" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-docs-connector/database/credentials/db-password"
curl -X PUT --data "admin" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-docs-connector/database/credentials/db-username"
