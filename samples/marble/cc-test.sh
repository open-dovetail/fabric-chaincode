#!/bin/bash
#
# Copyright (c) 2020, TIBCO Software Inc.
# All rights reserved.
#
# SPDX-License-Identifier: BSD-3-Clause-Open-MPI
#
# marble_cc tests executed from cli docker container of the Fabric test-network

. ./scripts/envVar.sh
setGlobals 1
export CORE_PEER_ADDRESS=peer0.org1.example.com:7051
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/test-network/organizations/peerOrganizations/org1.example.com/users/broker@org1.example.com/msp

ORDERER_ARGS="-o orderer.example.com:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA"
ORG1_ARGS="--peerAddresses peer0.org1.example.com:7051 --tlsRootCertFiles $PEER0_ORG1_CA"
ORG2_ARGS="--peerAddresses peer0.org2.example.com:9051 --tlsRootCertFiles $PEER0_ORG2_CA"

# insert test data
echo "insert 6 marbles by user 'broker' ..."
peer chaincode invoke $ORDERER_ARGS -C mychannel -n marble_cc $ORG1_ARGS $ORG2_ARGS -c '{"function":"initMarble","Args":["marble1","blue","35","tom"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n marble_cc $ORG1_ARGS $ORG2_ARGS -c '{"function":"initMarble","Args":["marble2","red","50","tom"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n marble_cc $ORG1_ARGS $ORG2_ARGS -c '{"function":"initMarble","Args":["marble3","blue","70","tom"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n marble_cc $ORG1_ARGS $ORG2_ARGS -c '{"function":"initMarble","Args":["marble4","purple","80","tom"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n marble_cc $ORG1_ARGS $ORG2_ARGS -c '{"function":"initMarble","Args":["marble5","purple","90","tom"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n marble_cc $ORG1_ARGS $ORG2_ARGS -c '{"function":"initMarble","Args":["marble6","purple","100","tom"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n marble_cc $ORG1_ARGS $ORG2_ARGS -c '{"function":"initMarble","Args":["marble7","purple","110","tom"]}'

# transfer marble ownership
echo "test transfer marbles by user 'tom' ..."
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/test-network/organizations/peerOrganizations/org1.example.com/users/tom@org1.example.com/msp
sleep 5
peer chaincode query -C mychannel -n marble_cc -c '{"Args":["readMarble","marble2"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n marble_cc $ORG1_ARGS $ORG2_ARGS -c '{"function":"transferMarble","Args":["marble2","jerry"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n marble_cc $ORG1_ARGS $ORG2_ARGS -c '{"function":"transferMarblesBasedOnColor","Args":["blue","jerry"]}'

echo "test range query ..."
sleep 5
peer chaincode query -C mychannel -n marble_cc -c '{"Args":["getMarblesByRange","marble1","marble5"]}'

# delete marble state, not history
echo "test delete and history"
peer chaincode invoke $ORDERER_ARGS -C mychannel -n marble_cc $ORG1_ARGS $ORG2_ARGS -c '{"function":"delete","Args":["marble1"]}'
sleep 5
peer chaincode query -C mychannel -n marble_cc -c '{"Args":["getHistory","marble1"]}'

# rich query
echo "test rich query by user 'jerry' ..."
setGlobals 2
export CORE_PEER_ADDRESS=peer0.org2.example.com:9051
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/test-network/organizations/peerOrganizations/org2.example.com/users/jerry@org2.example.com/msp

peer chaincode query -C mychannel -n marble_cc -c '{"Args":["queryMarblesByOwner","jerry"]}'

# query pagination using page-size and starting bookmark
echo "test pagination ..."
peer chaincode query -C mychannel -n marble_cc -c '{"Args":["getMarblesByRangeWithPagination","marble1","marble9", "3", ""]}'
peer chaincode query -C mychannel -n marble_cc -c '{"Args":["getMarblesByRangeWithPagination","marble1","marble9", "3", "marble5"]}'
peer chaincode query -C mychannel -n marble_cc -c '{"Args":["queryLargeMarblesWithPagination","60", "3", ""]}'
peer chaincode query -C mychannel -n marble_cc -c '{"Args":["queryLargeMarblesWithPagination","60", "3", "g1AAAAA-eJzLYWBgYMpgSmHgKy5JLCrJTq2MT8lPzkzJBYqz5yYWJeWkmoGkOWDSyBJZABz3EhU"]}'
