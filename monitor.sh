#!/bin/bash

set -e

sudo docker load -i ./tarpackage/monitserver.tar
sudo docker load -i ./tarpackage/cadvisor.tar

sleep 5

sudo docker run \
  --volume=/:/rootfs:ro \
  --volume=/var/run:/var/run:rw \
  --volume=/sys:/sys:ro \
  --volume=/var/lib/docker/:/var/lib/docker:ro \
  --publish=4194:8080 \
  --detach=true \
  google/cadvisor:latest

sleep 3

sudo docker run --net=host --privileged -d monitserver:latest

echo "Monitserver installation ok"
