#!/bin/bash

WSC_SERVICES=(carbonio-ws-collaboration)
./start-mails.sh "$@" "${WSC_SERVICES[@]}"