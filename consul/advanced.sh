
CONSUL_HTTP_ADDR="http://localhost:8500"

curl -X PUT --data "abq" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-abq/db-name"
curl -X PUT --data "carbonio_advanced" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-abq/db-username"
curl -X PUT --data "password" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-abq/db-password"

curl -X PUT --data "activesync" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-activesync/db-name"
curl -X PUT --data "carbonio_advanced" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-activesync/db-username"
curl -X PUT --data "password" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-activesync/db-password"

curl -X PUT --data "auth" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-auth/db-name"
curl -X PUT --data "carbonio_advanced" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-auth/db-username"
curl -X PUT --data "password" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-auth/db-password"

curl -X PUT --data "core" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-core/db-name"
curl -X PUT --data "carbonio_advanced" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-core/db-username"
curl -X PUT --data "password" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-core/db-password"

curl -X PUT --data "ha" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-ha/db-name"
curl -X PUT --data "carbonio_advanced" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-ha/db-username"
curl -X PUT --data "password" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-ha/db-password"

curl -X PUT --data "powerstore" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-powerstore/db-name"
curl -X PUT --data "carbonio_advanced" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-powerstore/db-username"
curl -X PUT --data "password" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-powerstore/db-password"

curl -X PUT --data "backup" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-backup/db-name"
curl -X PUT --data "carbonio_advanced" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-backup/db-username"
curl -X PUT --data "password" "$CONSUL_HTTP_ADDR/v1/kv/carbonio-backup/db-password"
