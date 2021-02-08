#!/bin/bash
#
# Copyright (c) 2020, TIBCO Software Inc.
# All rights reserved.
#
# SPDX-License-Identifier: BSD-3-Clause-Open-MPI
#
# build chaincode package from model.json.  package file will be in the same directory as the model.json
# usage:
#   ./build.sh ../samples/marble/marble.json marble_cc

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"; echo "$(pwd)")"

if [ "$#" -lt 1 ]; then
  echo "Usage: ./build.sh model-file [chaincode-name [version]]"
  exit 1
fi

MODEL_DIR="$(cd "$(dirname "$1")"; echo "$(pwd)")"
MODEL=${1##*/}

if [ "$#" -gt 1 ]; then
  CCNAME=$2
else
  CCNAME="${MODEL%.*}_cc"
fi

VERSION=1.0
if [ "$#" -gt 2 ]; then
  VERSION=$3
fi

# create and build source code
sed "s/{CCNAME}/${CCNAME}/" ${SCRIPT_DIR}/template.mod > ${MODEL_DIR}/go.mod
cd ${MODEL_DIR}
flogo create --cv v1.2.0 -f ${MODEL} -m go.mod ${CCNAME}
cd ${CCNAME}
flogo build --shim fabric_transaction --verbose
cd src
go mod tidy

# copy couchdb index
if [ -d "${MODEL_DIR}/META-INF" ]; then
  cp -rf ${MODEL_DIR}/META-INF .
fi

# construct chaincode package
cd ${MODEL_DIR}/${CCNAME}
if [ -f "bin/${CCNAME}" ]; then
  echo '{"path":"github.com/open-dovetail/fabric-chaincode/'${CCNAME}'","type":"golang","label":"'${CCNAME}_${VERSION}'"}' > metadata.json
  tar cfz code.tar.gz src
  tar cfz ${MODEL_DIR}/${CCNAME}_${VERSION}.tar.gz metadata.json code.tar.gz
  echo "Created chaincode package ${MODEL_DIR}/${CCNAME}_${VERSION}.tar.gz"
else
  echo "failed to build chaincode source code"
  exit 1
fi

# cleanup build files
if [ -f "${MODEL_DIR}/${CCNAME}_${VERSION}.tar.gz" ]; then
  echo "cleanup build files"
  rm -R ${MODEL_DIR}/${CCNAME}
  rm ${MODEL_DIR}/go.mod
fi
