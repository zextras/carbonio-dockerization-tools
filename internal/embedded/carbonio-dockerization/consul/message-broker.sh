CONSUL_HTTP_ADDR="http://localhost:8500"
curl -X PUT --data "guest" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-message-broker/default/username"
curl -X PUT --data "guest" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-message-broker/default/password"