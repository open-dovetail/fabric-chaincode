#!/bin/bash

# install and instantiate marble_cc in the Fabric test-network
# execute this script from the scripts folder of the cli docker container

. ./scripts/envVar.sh
ORDERER_ARGS="-o orderer.example.com:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA"

CCNAME=marble_cc
CC_PATH=${GOPATH}/src/github.com/chaincode
CDS_FILE=${CC_PATH}/${CCNAME}_1.0.tar.gz

if [ ! -f "${CDS_FILE}" ]; then
  echo "cannot find chaincode pacackage: ${CDS_FILE}"
  exit 1
fi

echo "org1 installs ${CCNAME}"
setGlobals 1
export CORE_PEER_ADDRESS=peer0.org1.example.com:7051
peer lifecycle chaincode install ${CDS_FILE}

echo "org2 installs ${CCNAME}"
setGlobals 2
export CORE_PEER_ADDRESS=peer0.org2.example.com:9051
peer lifecycle chaincode install ${CDS_FILE} >&log.txt
PACKAGE_ID=$(cat log.txt | grep "Chaincode code package identifier:" | sed 's/.*Chaincode code package identifier: //')

echo "org1 approves package ${PACKAGE_ID}"
setGlobals 1
export CORE_PEER_ADDRESS=peer0.org1.example.com:7051
peer lifecycle chaincode approveformyorg ${ORDERER_ARGS} --channelID mychannel --name ${CCNAME} --version 1.0 --package-id "${PACKAGE_ID}" --sequence 1
sleep 5

echo "org2 approves package ${PACKAGE_ID}"
setGlobals 2
export CORE_PEER_ADDRESS=peer0.org2.example.com:9051
peer lifecycle chaincode approveformyorg ${ORDERER_ARGS} --channelID mychannel --name ${CCNAME} --version 1.0 --package-id "${PACKAGE_ID}" --sequence 1
sleep 5

echo "commit chaincode ${CCNAME}"
PEER_CONN_PARMS="--peerAddresses peer0.org1.example.com:7051 --tlsRootCertFiles $PEER0_ORG1_CA --peerAddresses peer0.org2.example.com:9051 --tlsRootCertFiles $PEER0_ORG2_CA"
peer lifecycle chaincode commit ${ORDERER_ARGS} --channelID mychannel --name ${CCNAME} ${PEER_CONN_PARMS} --version 1.0 --sequence 1
