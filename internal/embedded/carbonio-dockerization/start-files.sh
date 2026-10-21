#!/bin/bash

FILES_SERVICES=(carbonio-files carbonio-docs-connector carbonio-docs-editor)
./start-mails.sh "$@" "${FILES_SERVICES[@]}"