#!/usr/bin/env bash

TAG=${1}

if [ -z "$TAG" ]; then
	TAG="community"
fi

# stop node
go run -tags=${TAG}  ../cmd/edge-node/main.go stop

# reference: https://pkg.go.dev/cmd/go/internal/test
go clean -testcache
go test -timeout 10s -tags="${TAG}" -cover ../...