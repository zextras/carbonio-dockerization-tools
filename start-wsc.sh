#!/bin/bash

WSC_SERVICES=(carbonio-ws-collaboration)
if [[ -z "$1" ]]; then
  ./start-mails.sh "${WSC_SERVICES[@]}"
else
  ./start-mails.sh "$1" "${WSC_SERVICES[@]}"
fi