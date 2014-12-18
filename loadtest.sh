#!/bin/bash

set -e

IP=`boot2docker ip`

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

siege -q -c 100 -i -t 2m --log=siege.log ${IP}:8888/helloworld/bob

fig stop
fig rm
