#!/bin/sh

CURRENT_PATH="$( cd "$( dirname "$(readlink -f $0)" )" && pwd )"
cd $CURRENT_PATH/src
export GOOS=linux
export GOARCH=amd64
exec go build -o $CURRENT_PATH/bin/vibenator
