#!/usr/bin/env bash

TARGET=../../EdgeAdmin/internal/serverconfigs
if [ -d ${TARGET} ]
then
	rm -rf ../../EdgeAdmin/internal/serverconfigs
fi
cp -R ../internal/configs/serverconfigs ../../EdgeAdmin/internal/configs/
cp -R ../internal/configs/serverconfigs ../../EdgeAPI/internal/configs