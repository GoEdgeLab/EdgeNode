#!/usr/bin/env bash

TAG=${1}

if [ -z "$TAG" ]; then
	TAG="community"
fi

go test -v ../... -tags=${TAG}