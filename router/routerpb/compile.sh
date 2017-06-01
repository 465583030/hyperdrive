#!/bin/bash

set -e

protoc --gofast_out="$(pwd)/router/routerpb/" \
  -I="$(pwd)/router/routerpb:${GOPATH}/src/github.com/gogo/protobuf:${GOPATH}/src/github.com/gogo/protobuf/protobuf" \
  "$(pwd)/router/routerpb/router.proto"
