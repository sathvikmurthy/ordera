#!/bin/bash

# Script to list all wallets from the Fabric ledger
# Uses the GetAllWallets chaincode function

set -e

echo "📋 Listing All Wallets from Fabric Network"
echo "==========================================="

# Configuration
CHAINCODE_NAME="wallet"
CHANNEL_NAME="mychannel"

# Get the path to fabric-samples
FABRIC_SAMPLES_PATH="/Users/sathvikcustiv/fabric-dev/fabric-samples"
TEST_NETWORK_PATH="${FABRIC_SAMPLES_PATH}/test-network"

# Check if test-network exists
if [ ! -d "$TEST_NETWORK_PATH" ]; then
    echo "❌ Error: test-network not found at $TEST_NETWORK_PATH"
    exit 1
fi

cd "$TEST_NETWORK_PATH"

# Set peer CLI environment variables for Org1
export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=$PWD/../config/
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051

echo ""
echo "🔍 Querying all wallets from the ledger..."
echo ""

# Query GetAllWallets function
RESULT=$(peer chaincode query \
    -C $CHANNEL_NAME \
    -n $CHAINCODE_NAME \
    -c '{"function":"GetAllWallets","Args":[]}' 2>&1)

# Check if query was successful
if [ $? -eq 0 ]; then
    echo "✅ Wallets retrieved successfully!"
    echo ""
    echo "Wallets:"
    echo "$RESULT" | jq '.' 2>/dev/null || echo "$RESULT"
else
    echo "❌ Failed to retrieve wallets"
    echo "$RESULT"
    exit 1
fi

echo ""
