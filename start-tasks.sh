#!/bin/bash

TASKS_SERVICES=(carbonio-tasks)
./start-mails.sh "$@" "${TASKS_SERVICES[@]}"