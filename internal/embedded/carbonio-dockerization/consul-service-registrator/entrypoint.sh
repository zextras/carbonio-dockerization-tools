#!/bin/sh

# Listen to Docker events and filter by 'start' and 'stop' events
docker events --filter "type=container" --filter "event=start" --filter "event=stop" --format "{{json .}}" |
while read event
do
    event_action=$(echo "$event" | jq -r '.Action')
    container_labels=$(echo "$event" | jq -r '.Actor.Attributes')

    if [ "$container_labels" != "{}" ]; then
        servicePort=$(echo "$container_labels" | jq -r '.servicePort // empty')
        if [ -z "${servicePort}" ]; then
          continue
        fi;
        serviceId=$(echo "$container_labels" | jq -r '.serviceId // empty')
        serviceName=$(echo "$container_labels" | jq -r '.serviceName // empty')
        serviceNameFallback=$(echo "$container_labels" | jq -r '."com.docker.compose.service"')
        SERVICE_NAME=${serviceName:-${serviceNameFallback}}
        SERVICE_ID=${serviceId:-${SERVICE_NAME}}

        if [ "$event_action" = "start" ]; then
            echo "Container '${SERVICE_NAME}' started."
            echo "Port: ${servicePort}"
            curl -X PUT -H "Content-Type: application/json" -d "{
              \"id\": \"${SERVICE_ID}\",
              \"name\": \"${SERVICE_NAME}\",
              \"address\": \"${SERVICE_NAME}\",
              \"port\": ${servicePort},
              \"tags\": [
                \"traefik.enable=true\",
                \"traefik.http.routers.${SERVICE_NAME}.entrypoints=${SERVICE_NAME}\",
                \"traefik.http.routers.${SERVICE_NAME}.rule=PathPrefix(\\\"/\\\")\",
                \"traefik.http.services.${SERVICE_NAME}.loadbalancer.server.port=${servicePort}\"
              ],
              \"check\": {
                \"tcp\": \"${SERVICE_NAME}:${servicePort}\",
                \"interval\": \"10s\",
                \"timeout\": \"3s\"
              }
            }" http://consul:8500/v1/agent/service/register
        elif [ "$event_action" = "stop" ]; then
            echo "Container '$serviceName' stopped. Deregistering from Consul."
            curl -X PUT http://consul:8500/v1/agent/service/deregister/"${SERVICE_ID}"
        fi
    fi
done
