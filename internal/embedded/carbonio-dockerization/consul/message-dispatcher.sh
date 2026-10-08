
CONSUL_HTTP_ADDR="http://localhost:8500"
curl -X PUT --data "carbonio-message-dispatcher-db" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-message-dispatcher-db/db-name"
curl -X PUT --data "admin" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-message-dispatcher-db/db-username"
curl -X PUT --data "admin" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-message-dispatcher-db/db-password"
