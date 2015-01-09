#!/bin/bash

set -e

IP=`boot2docker ip`
SIEGE="siege -q -c 100 -i -t 2m --log=siege.log ${IP}:8888/helloworld/bob"
if [ -z $IP ]; then
  echo "I can't find the boot2docker ip! Make sure boot2docker is running first!"
  exit 1
fi

fig up -d registrator

#fig up -d --no-recreate registrator
fig up -d helloworld
fig scale helloworld=4

curl -sS -X PUT -d '/helloworld' ${IP}:8500/v1/kv/conductor-services/helloworld > /dev/null

fig up -d conductor
if [ "x$1" == "x--scale" ]; then
  echo "Starting siege with scaling test mode..."
  $SIEGE &
  sleep 20s
  fig scale helloworld=6
  sleep 40s
  fig scale helloworld=2
  sleep 15s
  fig scale helloworld=4
  fg %%
else
  echo -n "Starting siege..."
  $SIEGE
fi

fig stop
fig rm
