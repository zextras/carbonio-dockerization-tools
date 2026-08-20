#!/bin/bash

WSC_SERVICES=(carbonio-ws-collaboration carbonio-videoserver)
./start-mails.sh "$@" "${WSC_SERVICES[@]}"