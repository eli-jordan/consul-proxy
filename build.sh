#!/bin/bash

glide install
gox -osarch="linux/amd64 linux/386 darwin/amd64 darwin/386 windows/amd64 windows/386" -output="binaries/{{.OS}}_{{.Arch}}/{{.Dir}}"
