#!/bin/bash

# Script to deploy the priority wallet chaincode to Hyperledger Fabric test-network
# This script packages, installs, approves, and commits the chaincode

set -e

echo "🚀 Deploying Priority Wallet Chaincode to Fabric Network"
echo "=========================================================="

# Configuration
CHAINCODE_NAME="wallet"
CHAINCODE_VERSION="1.5"
CHANNEL_NAME="mychannel"
CHAINCODE_SEQUENCE=6
CC_SRC_PATH="/Users/sathvikcustiv/fabric-dev/priority-fabric-project/chaincode"

# Get the absolute path to fabric-samples
FABRIC_SAMPLES_PATH="/Users/sathvikcustiv/fabric-dev/fabric-samples"
TEST_NETWORK_PATH="${FABRIC_SAMPLES_PATH}/test-network"

# Check if test-network exists
if [ ! -d "$TEST_NETWORK_PATH" ]; then
    echo "❌ Error: test-network not found at $TEST_NETWORK_PATH"
    exit 1
fi

cd "$TEST_NETWORK_PATH"

# Step 1: Check if network is running
echo ""
echo "📡 Step 1: Checking if Fabric network is running..."
if ! docker ps | grep -q "peer0.org1.example.com"; then
    echo "⚠️  Network is not running. Starting test-network..."
    ./network.sh down
    ./network.sh up createChannel -c $CHANNEL_NAME -ca
    echo "✅ Network started successfully"
else
    echo "✅ Network is already running"
fi

# Step 2: Package the chaincode
echo ""
echo "📦 Step 2: Packaging chaincode..."
cd "$TEST_NETWORK_PATH"

# Set peer CLI environment variables for Org1
export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=$PWD/../config/
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051

# Package chaincode
peer lifecycle chaincode package ${CHAINCODE_NAME}.tar.gz \
    --path ${CC_SRC_PATH} \
    --lang golang \
    --label ${CHAINCODE_NAME}_${CHAINCODE_VERSION}

echo "✅ Chaincode packaged: ${CHAINCODE_NAME}.tar.gz"

# Step 3: Install chaincode on Org1 peer
echo ""
echo "📥 Step 3: Installing chaincode on Org1 peer..."
peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz
echo "✅ Chaincode installed on Org1"

# Step 4: Install chaincode on Org2 peer
echo ""
echo "📥 Step 4: Installing chaincode on Org2 peer..."
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051

peer lifecycle chaincode install ${CHAINCODE_NAME}.tar.gz
echo "✅ Chaincode installed on Org2"

# Step 5: Query installed chaincode to get package ID
echo ""
echo "🔍 Step 5: Querying installed chaincode..."
CC_PACKAGE_ID=$(peer lifecycle chaincode queryinstalled | grep ${CHAINCODE_NAME}_${CHAINCODE_VERSION} | sed -n 's/^Package ID: //; s/, Label:.*$//; p' | head -1)

if [ -z "$CC_PACKAGE_ID" ]; then
    echo "❌ Error: Could not find package ID"
    exit 1
fi

echo "✅ Package ID: $CC_PACKAGE_ID"

# Step 6: Approve chaincode for Org2
echo ""
echo "✅ Step 6: Approving chaincode for Org2..."
peer lifecycle chaincode approveformyorg \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --channelID $CHANNEL_NAME \
    --name $CHAINCODE_NAME \
    --version $CHAINCODE_VERSION \
    --package-id $CC_PACKAGE_ID \
    --sequence $CHAINCODE_SEQUENCE \
    --tls \
    --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem

echo "✅ Chaincode approved for Org2"

# Step 7: Approve chaincode for Org1
echo ""
echo "✅ Step 7: Approving chaincode for Org1..."
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_ADDRESS=localhost:7051

peer lifecycle chaincode approveformyorg \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --channelID $CHANNEL_NAME \
    --name $CHAINCODE_NAME \
    --version $CHAINCODE_VERSION \
    --package-id $CC_PACKAGE_ID \
    --sequence $CHAINCODE_SEQUENCE \
    --tls \
    --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem

echo "✅ Chaincode approved for Org1"

# Step 8: Check commit readiness
echo ""
echo "🔍 Step 8: Checking commit readiness..."
peer lifecycle chaincode checkcommitreadiness \
    --channelID $CHANNEL_NAME \
    --name $CHAINCODE_NAME \
    --version $CHAINCODE_VERSION \
    --sequence $CHAINCODE_SEQUENCE \
    --tls \
    --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
    --output json

# Step 9: Commit chaincode
echo ""
echo "🚀 Step 9: Committing chaincode to channel..."
peer lifecycle chaincode commit \
    -o localhost:7050 \
    --ordererTLSHostnameOverride orderer.example.com \
    --channelID $CHANNEL_NAME \
    --name $CHAINCODE_NAME \
    --version $CHAINCODE_VERSION \
    --sequence $CHAINCODE_SEQUENCE \
    --tls \
    --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
    --peerAddresses localhost:7051 \
    --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
    --peerAddresses localhost:9051 \
    --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt

echo "✅ Chaincode committed successfully"

# Step 10: Query committed chaincode
echo ""
echo "🔍 Step 10: Verifying chaincode deployment..."
peer lifecycle chaincode querycommitted \
    --channelID $CHANNEL_NAME \
    --name $CHAINCODE_NAME \
    --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem

echo ""
echo "🎉 SUCCESS! Chaincode deployed successfully!"
echo ""
echo "📋 Deployment Summary:"
echo "   Chaincode Name: $CHAINCODE_NAME"
echo "   Version: $CHAINCODE_VERSION"
echo "   Channel: $CHANNEL_NAME"
echo "   Package ID: $CC_PACKAGE_ID"
echo ""
echo "💡 Next Steps:"
echo "   1. Start your gateway server: go run . --use-fabric"
echo "   2. Submit transactions via HTTP API"
echo "   3. Monitor the logs to see transactions being committed to the ledger"
echo ""
