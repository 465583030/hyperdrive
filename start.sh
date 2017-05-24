#!/usr/bin/env /bin/bash

set -e

ID=$1
if [ -z $ID ]
then
  ID=1
fi

pushd build

./hyperdrive -cluster "http://127.0.0.1:8211,http://127.0.0.1:8212,http://127.0.0.1:8213" -id $ID -port "831${ID}" -api-port "841${ID}"

popd
