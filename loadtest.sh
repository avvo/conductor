#!/bin/bash

set -e

IP=`docker-machine ip`
SIEGE="siege -q -c 100 -i -t 2m --log=siege.log ${IP}:8888/helloworld/bob"
if [ -z $IP ]; then
  echo "I can't find the docker-machine ip! Make sure docker is running first!"
  exit 1
fi

docker-compose up -d registrator

#fig up -d --no-recreate registrator
docker-compose up -d helloworld
docker-compose scale helloworld=4

curl -sS -X PUT -d '/helloworld' ${IP}:8500/v1/kv/conductor/services/helloworld > /dev/null

docker-compose up -d conductor
if [ "x$1" == "x--scale" ]; then
  { echo "Starting siege with scaling test mode..."; $SIEGE ; } &
  sleep 20s
  docker-compose scale helloworld=6
  sleep 20s
  docker-compose scale helloworld=1
  sleep 20s
  docker-compose scale helloworld=4
  sleep 20s
  docker-compose scale helloworld=3
  sleep 20s
  docker-compose scale helloworld=6
  wait
else
  echo -n "Starting siege..."
  $SIEGE
fi

docker-compose stop
docker-compose rm
