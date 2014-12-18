#!/bin/bash

set -e

IP=`boot2docker ip`

if [ -z $IP ]; then
  echo "I can't find the boot2docker ip! Make sure boot2docker is running first!"
  exit 1
fi

fig up -d consul
# Let consul figure out who is master before continuing
sleep 1

#fig up -d --no-recreate registrator
fig up -d helloworld
fig scale helloworld=4

curl -sS -X PUT -d '/helloworld' ${IP}:8500/v1/kv/conductor-services/helloworld > /dev/null

fig up -d --no-recreate conductor

docker logs conductor_conductor_1

siege -q -c 10 -b -t 2m --log=seige.log ${IP}:8888/helloworld/

fig stop

fig rm
