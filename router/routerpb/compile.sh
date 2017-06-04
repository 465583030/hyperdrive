#!/bin/bash

set -e

go get golang.org/x/tools/cmd/goimports

DIRS="router/routerpb"

for dir in ${DIRS}; do
  pushd $dir
    protoc --gofast_out="$(pwd)" \
      -I="$(pwd)/:${GOPATH}/src/github.com/gogo/protobuf:${GOPATH}/src/github.com/gogo/protobuf/protobuf" \
      "$(pwd)/router.proto"

    sed -i.bak -E "s/github\.com\/coreos\/(gogoproto|github\.com|golang\.org|google\.golang\.org)/\1/g" *.pb.go
    sed -i.bak -E 's/import _ \"gogoproto\"//g' *.pb.go
    rm -f *.bak
    goimports -w *.pb.go
  popd
done
