#!/bin/bash

FILES_SERVICES=(carbonio-files carbonio-docs-connector carbonio-docs-editor)
if [[ -z "$1" ]]; then
  ./start-mails.sh "${FILES_SERVICES[@]}"
else
  ./start-mails.sh "$1" "${FILES_SERVICES[@]}"
fi