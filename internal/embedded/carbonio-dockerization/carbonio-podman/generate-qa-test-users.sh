#!/bin/bash
domain="carbonio.localhost"
password="812feee9-c08e-4ec5-9734-07f6e7a6b096"
for i in $(seq 100 200); do
  echo "ca test$i@$domain $password cn \"Test $i\" displayName \"Test $i\""
done