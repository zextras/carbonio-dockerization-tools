
CONSUL_HTTP_ADDR="http://localhost:8500"
curl -X PUT --data "carbonio-tasks-db" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-tasks/database/credentials/db-name"
curl -X PUT --data "admin" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-tasks/database/credentials/db-password"
curl -X PUT --data "admin" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-tasks/database/credentials/db-username"
