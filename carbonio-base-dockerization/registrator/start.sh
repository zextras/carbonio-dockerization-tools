
SERVICE_IP=$(hostname -i)
SERVICE_CHECK_TYPE="${SERVICE_CHECK_TYPE:-tcp}"

if [ "$SERVICE_CHECK_TYPE" = "http" ]; then
    SERVICE_CHECK="http://${SERVICE_IP}:${SERVICE_PORT}${SERVICE_CHECK_PATH}"
else
    SERVICE_CHECK="${SERVICE_IP}:${SERVICE_PORT}"
fi

# Build JSON
SERVICE_JSON=$(cat <<EOF
{
  "name": "$SERVICE_NAME",
  "address": "$SERVICE_IP",
  "port": $SERVICE_PORT,
  "check": {
        "$SERVICE_CHECK_TYPE": "$SERVICE_CHECK",
        "interval": "10s",
        "timeout": "3s"
      }
}
EOF
)
echo "Registering configuration: ${SERVICE_JSON}"
# Register in Consul
curl -X PUT --data "$SERVICE_JSON" "$CONSUL_HTTP_ADDR/v1/agent/service/register"