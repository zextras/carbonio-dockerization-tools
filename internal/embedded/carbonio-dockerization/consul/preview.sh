
CONSUL_HTTP_ADDR="http://localhost:8500"
curl -X PUT --data "carbonio-preview-db" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-preview/database/credentials/db-name"
curl -X PUT --data "admin" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-preview/database/credentials/db-password"
curl -X PUT --data "admin" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-preview/database/credentials/db-username"
