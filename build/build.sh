#!/usr/bin/env bash

function build() {
	VERSION_DATA=$(cat ../internal/const/const.go)
	re="Version[ ]+=[ ]+\"([0-9.]+)\""
	if [[ $VERSION_DATA =~ $re ]]; then
		VERSION=${BASH_REMATCH[1]}
	else
		echo "could not match version"
		exit
	fi

	echo "checking ..."
	ZIP_PATH=$(which zip)
	if [ -z $ZIP_PATH ]; then
		echo "we need 'zip' command to compress files"
		exit
	fi

	echo "building v${VERSION}/${1}/${2} ..."
	NAME="edge-node"
	DIST="../dist/${NAME}"
	ZIP="${NAME}-${1}-${2}-v${VERSION}.zip"

	echo "copying ..."
	if [ ! -d $DIST ]; then
		mkdir $DIST
		mkdir $DIST/bin
		mkdir $DIST/configs
		mkdir $DIST/logs
	fi

	cp configs/api.template.yaml $DIST/configs
	cp -R www $DIST/

	echo "building ..."
	env GOOS=${1} GOARCH=${2} go build -o $DIST/bin/${NAME} -ldflags="-s -w" ../cmd/edge-node/main.go

	echo "zip files"
	cd "${DIST}/../" || exit
	if [ -f "${ZIP}" ]; then
		rm -f "${ZIP}"
	fi
	zip -r -X -q "${ZIP}" ${NAME}/
	rm -rf ${NAME}
	cd - || exit

	echo "OK"
}

build "linux" "amd64"