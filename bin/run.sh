#!/usr/bin/env bash

cd /go/src/github.com/goforbroke1006/dataoverseersvc/
#dep ensure -v
go run cmd/dataoverseersvc/main.go -log-file=./daemon.log -сfg-file=./config-docker.yml