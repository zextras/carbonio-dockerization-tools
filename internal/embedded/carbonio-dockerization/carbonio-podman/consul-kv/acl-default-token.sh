CONSUL_HTTP_ADDR="http://localhost:8500"

# Create a read-only policy for the default agent token
curl -sf -X PUT -H "X-Consul-Token: $CONSUL_HTTP_TOKEN" "$CONSUL_HTTP_ADDR/v1/acl/policy" -d '{
  "Name": "default-read",
  "Description": "Read-only access to services and nodes",
  "Rules": "node_prefix \"\" { policy = \"read\" }\nservice_prefix \"\" { policy = \"read\" }"
}'

# Create a token with a known secret ID and attach the policy
curl -sf -X PUT -H "X-Consul-Token: $CONSUL_HTTP_TOKEN" "$CONSUL_HTTP_ADDR/v1/acl/token" -d '{
  "SecretID": "11111111-1111-1111-1111-111111111111",
  "Description": "Default agent read-only token",
  "Policies": [{"Name": "default-read"}]
}'
