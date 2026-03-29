#!/bin/bash

# Script to add balance to a wallet
# Usage: ./add-balance.sh WALLET_ADDRESS AMOUNT

set -e

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 WALLET_ADDRESS AMOUNT"
    echo "Example: $0 dec67253a3a6a13935f47dec7fb2a2c16b481bb9 1000"
    exit 1
fi

WALLET_ADDRESS=$1
AMOUNT=$2

echo "💰 Adding Balance to Wallet"
echo "============================"
echo "Wallet: $WALLET_ADDRESS"
echo "Amount: $AMOUNT"
echo ""

# Configuration
CHAINCODE_NAME="wallet"
CHANNEL_NAME="mychannel"

# Get the script directory and project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Get the path to fabric-samples
FABRIC_SAMPLES_PATH="${PROJECT_ROOT}/../fabric-samples"
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

echo "🔄 Invoking AddBalance function..."
echo ""

# Invoke AddBalance function
peer chaincode invoke \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --tls \
    --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
    -C $CHANNEL_NAME \
    -n $CHAINCODE_NAME \
    --peerAddresses localhost:7051 \
    --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
    --peerAddresses localhost:9051 \
    --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
    -c "{\"function\":\"AddBalance\",\"Args\":[\"$WALLET_ADDRESS\",\"$AMOUNT\"]}"

echo ""
echo "✅ Balance added successfully!"
echo ""
echo "🔍 Querying wallet to verify..."
echo ""

# Query the wallet
WALLET_DATA=$(peer chaincode query \
    -C $CHANNEL_NAME \
    -n $CHAINCODE_NAME \
    -c "{\"function\":\"GetWallet\",\"Args\":[\"$WALLET_ADDRESS\"]}")

echo "Wallet Data:"
echo "$WALLET_DATA" | jq '.'

echo ""
echo "✅ Done!"
echo ""
