#!/bin/bash
# setup dev environment for open-dovetail chaincode build and test

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"; echo "$(pwd)")"

function checkGo {
  which go
  if [ "$?" -ne 0 ]; then
    echo "Please install Go from https://golang.org/dl/"
    exit 1
  fi
  if [ -z "$GOPATH" ]; then
    echo "Set GOPATH to $HOME/go"
    mkdir -p ${HOME}/go/bin
    export GOPATH=${HOME}/go
  fi
  echo $PATH | grep $GOPATH
  if [ "$?" -ne 0 ]; then
    echo "Add ${GOPATH}/bin to PATH"
    export PATH=${PATH}:${GOPATH}/bin
  fi
  go version
}

function installFlogo {
  flogo create -h | grep mod
  if [ "$?" -ne 0 ]; then
    echo "install Flogo CLI"
    cd ${SCRIPT_DIR}/tools
    go install github.com/project-flogo/cli/cmd/flogo
    flogo create -h | grep mod
    if [ "$?" -ne 0 ]; then
      echo "Failed to install Flog CLI"
      exit 1
    fi
  fi
  flogo --version
}

function installFabricSample {
  local hlf_path=${SCRIPT_DIR}/../../hyperledger
  if [ ! -d "${hlf_path}/fabric-samples" ]; then
    echo "download Hyperledger Fabric samples to ${hlf_path} ..."
    mkdir -p ${hlf_path}
    cd ${hlf_path}
    curl -sSL http://bit.ly/2ysbOFE | bash -s -- 2.2.1 1.4.9

    echo "setup cli container for test-network"
    cp ${SCRIPT_DIR}/docker-compose-cli.yaml ${hlf_path}/fabric-samples/test-network/docker
    sed -i -e "s/COMPOSE_FILE_BASE=docker\/docker-compose-test-net.yaml.*/COMPOSE_FILE_BASE=\"docker\/docker-compose-test-net.yaml -f docker\/docker-compose-cli.yaml\"/" ${hlf_path}/fabric-samples/test-network/network.sh
  fi
  echo "Hyperledger Fabric samples are in ${hlf_path}/fabric-samples"
}

checkGo
installFlogo
installFabricSample