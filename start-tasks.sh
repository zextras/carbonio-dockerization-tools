#!/bin/bash

TASKS_SERVICES=(carbonio-tasks)
if [[ -z "$1" ]]; then
  ./start-mails.sh "${TASKS_SERVICES[@]}"
else
  ./start-mails.sh "$1" "${TASKS_SERVICES[@]}"
fi