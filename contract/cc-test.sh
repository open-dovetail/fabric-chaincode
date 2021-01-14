#!/bin/bash
#
# Copyright (c) 2020, TIBCO Software Inc.
# All rights reserved.
#
# SPDX-License-Identifier: BSD-3-Clause-Open-MPI
#
# sample_cc tests executed from cli docker container of the Fabric test-network

. ./scripts/envVar.sh
CCNAME=sample_cc

setGlobals 1
export CORE_PEER_ADDRESS=peer0.org1.example.com:7051
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/test-network/organizations/peerOrganizations/org1.example.com/users/broker@org1.example.com/msp

ORDERER_ARGS="-o orderer.example.com:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA"
ORG1_ARGS="--peerAddresses peer0.org1.example.com:7051 --tlsRootCertFiles $PEER0_ORG1_CA"
ORG2_ARGS="--peerAddresses peer0.org2.example.com:9051 --tlsRootCertFiles $PEER0_ORG2_CA"

# insert test data
echo "insert 6 marbles by user 'broker' ..."
peer chaincode invoke $ORDERER_ARGS -C mychannel -n $CCNAME $ORG1_ARGS $ORG2_ARGS -c '{"function":"createMarble","Args":["marble1","blue","35","tom"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n $CCNAME $ORG1_ARGS $ORG2_ARGS -c '{"function":"createMarble","Args":["marble2","red","50","tom"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n $CCNAME $ORG1_ARGS $ORG2_ARGS -c '{"function":"createMarble","Args":["marble3","blue","70","tom"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n $CCNAME $ORG1_ARGS $ORG2_ARGS -c '{"function":"createMarble","Args":["marble4","purple","80","tom"]}'

# transfer marble ownership
echo "test transfer marbles by user 'tom' ..."
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/test-network/organizations/peerOrganizations/org1.example.com/users/tom@org1.example.com/msp
sleep 5
peer chaincode invoke $ORDERER_ARGS -C mychannel -n $CCNAME $ORG1_ARGS $ORG2_ARGS -c '{"function":"transferMarble","Args":["marble2","jerry"]}'

echo "test transient request data to insert on private data collection"
MARBLE=$(echo -n "{\"name\":\"marble1\",\"price\":99}" | base64 | tr -d \\n)
peer chaincode invoke $ORDERER_ARGS -C mychannel -n $CCNAME $ORG1_ARGS $ORG2_ARGS -c '{"function":"offerPrice","Args":[]}' --transient "{\"marble\":\"$MARBLE\"}"
echo "test composite-key query and bulk update"
peer chaincode invoke $ORDERER_ARGS -C mychannel -n $CCNAME $ORG1_ARGS $ORG2_ARGS -c '{"function":"transferMarblesBasedOnColor","Args":["blue","jerry"]}'

# get history
echo "test get history ..."
sleep 5
peer chaincode query -C mychannel -n $CCNAME -c '{"Args":["getHistory","marble1"]}'

echo "test private data hash"
peer chaincode query -C mychannel -n $CCNAME -c '{"Args":["getMarblePrice","marble1"]}'

# rich query
echo "test rich query by user 'jerry' ..."
setGlobals 2
export CORE_PEER_ADDRESS=peer0.org2.example.com:9051
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/test-network/organizations/peerOrganizations/org2.example.com/users/jerry@org2.example.com/msp

peer chaincode query -C mychannel -n $CCNAME -c '{"Args":["queryMarblesByOwner","jerry"]}'
