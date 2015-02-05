#!/bin/bash

VERSION=$1

if [ "x$VERSION" == "x" ];
then
	docker build --rm -t conductor .
	docker tag conductor registry.docker.prod.avvo.com/conductor
	docker push registry.docker.prod.avvo.com/conductor
else
	docker build --rm -t conductor:$VERSION .
	docker tag conductor:$VERSION registry.docker.prod.avvo.com/conductor:$VERSION
	docker push registry.docker.prod.avvo.com/conductor
fi
