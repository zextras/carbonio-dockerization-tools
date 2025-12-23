
CONSUL_HTTP_ADDR="http://localhost:8500"
curl -X PUT --data "carbonio-files-db" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-files/db-name"
curl -X PUT --data "admin" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-files/db-password"
curl -X PUT --data "admin" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-files/db-username"
curl -X PUT --data "999999999" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-docs-connector/max-file-size-in-mb/.*"
