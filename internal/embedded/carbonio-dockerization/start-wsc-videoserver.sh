#!/bin/bash

# the browser can't reach the Docker network, so Janus must advertise a host IP
if [[ "$(uname)" == "Darwin" ]]; then
    export NAT_IP="${NAT_IP:-$(ipconfig getifaddr "$(route -n get default | awk '/interface:/{print $2}')")}"
fi

WSC_SERVICES=(carbonio-ws-collaboration carbonio-videoserver)
./start-mails.sh "$@" "${WSC_SERVICES[@]}"
