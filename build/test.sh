#!/usr/bin/env bash

TAG=${1}

if [ -z "$TAG" ]; then
	TAG="community"
fi

# reference: https://pkg.go.dev/cmd/go/internal/test
go clean -testcache
go test -timeout 10s -tags="${TAG}" -cover ../...