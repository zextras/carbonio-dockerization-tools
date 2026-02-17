
CONSUL_HTTP_ADDR="http://localhost:8500"
curl -X PUT --data "carbonio-videorecorder-db" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-videorecorder-db/db-name"
curl -X PUT --data "admin" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-videorecorder-db/db-username"
curl -X PUT --data "admin" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-videorecorder-db/db-password"
