#!/bin/bash

# Script to create wallets using Fabric peer CLI
# This creates wallets by directly invoking the CreateWallet chaincode function

set -e

echo "🔧 Creating Wallets on Fabric Network"
echo "======================================"

# Configuration
CHAINCODE_NAME="wallet"
CHANNEL_NAME="mychannel"
NUM_WALLETS=${1:-2}  # Default to 2 wallets if not specified

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

echo ""
echo "📝 Creating $NUM_WALLETS wallet(s)..."
echo ""

# Array to store created wallet addresses
declare -a WALLET_ADDRESSES

# Create wallets
for i in $(seq 1 $NUM_WALLETS); do
    echo "Creating wallet #$i..."
    
    # Invoke CreateWallet function
    echo "Invoking CreateWallet chaincode function..."
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
        -c '{"function":"CreateWallet","Args":[]}'
    
    # Query the wallet to get the address (since we can't easily parse invoke output)
    # We'll query all wallets and get the latest one
    echo "Querying created wallet..."
    WALLETS=$(peer chaincode query \
        -C $CHANNEL_NAME \
        -n $CHAINCODE_NAME \
        -c '{"function":"GetAllWallets","Args":[]}')
    
    # Extract the last wallet address from the JSON array
    WALLET_ADDRESS=$(echo "$WALLETS" | jq -r '.[-1].Address' 2>/dev/null)
    
    if [ -z "$WALLET_ADDRESS" ] || [ "$WALLET_ADDRESS" = "null" ]; then
        echo "⚠️  Could not retrieve wallet address automatically"
        echo "Please run: ./list-wallets.sh to see all wallets"
    else
        WALLET_ADDRESSES+=("$WALLET_ADDRESS")
        echo "✅ Wallet #$i created: $WALLET_ADDRESS"
    fi
    
    echo ""
done

# Print summary
echo "======================================"
echo "🎉 Wallet Creation Summary"
echo "======================================"
echo "Total wallets created: ${#WALLET_ADDRESSES[@]}"
echo ""
echo "Wallet Addresses:"
for i in "${!WALLET_ADDRESSES[@]}"; do
    echo "  Wallet $((i+1)): ${WALLET_ADDRESSES[$i]}"
done

echo ""
echo "💡 Save these addresses for your transactions!"
echo ""
echo "Example transaction:"
echo "curl -X POST http://localhost:8080/submit \\"
echo "  -H \"Content-Type: application/json\" \\"
echo "  -d '{\"from\": \"${WALLET_ADDRESSES[0]}\", \"to\": \"${WALLET_ADDRESSES[1]}\", \"amount\": \"100\", \"txType\": \"transfer\"}'"
echo ""
