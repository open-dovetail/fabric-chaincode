#!/bin/bash
#
# Copyright (c) 2020, TIBCO Software Inc.
# All rights reserved.
#
# SPDX-License-Identifier: BSD-3-Clause-Open-MPI
#
# sample_cc tests executed from cli docker container of the Fabric test-network

. ./scripts/envVar.sh
CCNAME=simple_cc

setGlobals 1
export CORE_PEER_ADDRESS=peer0.org1.example.com:7051
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/test-network/organizations/peerOrganizations/org1.example.com/users/User1@org1.example.com/msp

ORDERER_ARGS="-o orderer.example.com:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile $ORDERER_CA"
ORG1_ARGS="--peerAddresses peer0.org1.example.com:7051 --tlsRootCertFiles $PEER0_ORG1_CA"
ORG2_ARGS="--peerAddresses peer0.org2.example.com:9051 --tlsRootCertFiles $PEER0_ORG2_CA"

# create test data
echo "create 4 marbles by user 'User1' ..."
peer chaincode invoke $ORDERER_ARGS -C mychannel -n $CCNAME $ORG1_ARGS $ORG2_ARGS -c '{"function":"createMarble","Args":["marble1","blue","35","tom"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n $CCNAME $ORG1_ARGS $ORG2_ARGS -c '{"function":"createMarble","Args":["marble2","red","50","tom"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n $CCNAME $ORG1_ARGS $ORG2_ARGS -c '{"function":"createMarble","Args":["marble3","blue","70","tom"]}'
peer chaincode invoke $ORDERER_ARGS -C mychannel -n $CCNAME $ORG1_ARGS $ORG2_ARGS -c '{"function":"createMarble","Args":["marble4","purple","80","tom"]}'
sleep 5

# get marble
echo "test get marble1 ..."
peer chaincode query -C mychannel -n $CCNAME -c '{"Args":["getMarble","marble1"]}'
echo "test get marble2 ..."
peer chaincode query -C mychannel -n $CCNAME -c '{"Args":["getMarble","marble2"]}'
