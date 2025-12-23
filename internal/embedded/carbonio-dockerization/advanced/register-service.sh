#!/bin/bash

registerService() {
  serviceId=$1
  serviceName=$2
  serviceAddress=$3
  servicePort=$4
curl -X PUT -H "Content-Type: application/json" -d "{
              \"id\": \"${serviceId}\",
              \"name\": \"${serviceName}\",
              \"address\": \"${serviceAddress}\",
              \"port\": ${servicePort},
              \"tags\": [
                \"traefik.enable=true\",
                \"traefik.http.routers.${serviceName}.entrypoints=${serviceName}\",
                \"traefik.http.routers.${serviceName}.rule=PathPrefix(\\\"/\\\")\",
                \"traefik.http.services.${serviceName}.loadbalancer.server.port=${servicePort}\"
              ],
              \"check\": {
                \"tcp\": \"${serviceName}:${servicePort}\",
                \"interval\": \"10s\",
                \"timeout\": \"3s\"
              }
            }" http://consul:8500/v1/agent/service/register
}

unregister() {
  curl -X PUT http://consul:8500/v1/agent/service/deregister/"${HOSTNAME}-advanced"
  curl -X PUT http://consul:8500/v1/agent/service/deregister/"${HOSTNAME}-auth"
  curl -X PUT http://consul:8500/v1/agent/service/deregister/"${HOSTNAME}-address-book"
}

register() {
  # TODO: find a way to register a dynamic address.
  # Using static address carbonio-mailbox, because mailbox has strict rules about hostnames
  # docker.carbonio.localhost is only reachable by proxy because it has the same .localhost subdomain
  # Consul instead cannot find it and so are not able other services
  touch /tmp/post_start.log
  echo "Registering advanced..." > /tmp/post_start.log
  registerService "${HOSTNAME}-advanced" "carbonio-advanced" "carbonio-mailbox" "8742"
  registerService "${HOSTNAME}-auth" "carbonio-auth" "carbonio-mailbox" "8742"
  registerService "${HOSTNAME}-address-book" "carbonio-address-book" "carbonio-mailbox" "8388"
  echo "Advanced registered" >> /tmp/post_start.log
}
