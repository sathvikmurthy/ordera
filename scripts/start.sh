#!/bin/bash

# Quick start script for Priority Fabric Transaction Gateway
# This script helps you get started quickly with the system

set -e

echo "🚀 Priority Fabric Transaction Gateway - Quick Start"
echo "====================================================="
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored messages
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Get the script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Check if we're in the right directory
if [ ! -f "$PROJECT_ROOT/cmd/server.go" ]; then
    print_error "Please run this script from the priority-fabric-project/scripts directory"
    exit 1
fi

cd "$PROJECT_ROOT"

# Parse command line arguments
DEPLOY_CHAINCODE=false
USE_FABRIC=false
PORT=""
BATCH_SIZE=""
BATCH_TIMEOUT=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --deploy)
            DEPLOY_CHAINCODE=true
            shift
            ;;
        --fabric)
            USE_FABRIC=true
            shift
            ;;
        --port)
            PORT="$2"
            shift 2
            ;;
        --batch-size)
            BATCH_SIZE="$2"
            shift 2
            ;;
        --batch-timeout)
            BATCH_TIMEOUT="$2"
            shift 2
            ;;
        --help)
            echo "Usage: ./start.sh [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --deploy          Deploy chaincode to Fabric network before starting"
            echo "  --fabric          Connect to Fabric network (default: simulation mode)"
            echo "  --port PORT       HTTP server port (default: 8080)"
            echo "  --batch-size N    Transactions per batch (default: 100)"
            echo "  --batch-timeout T Batch timeout (default: 2s)"
            echo "  --help            Show this help message"
            echo ""
            echo "Examples:"
            echo "  ./start.sh                           # Start in simulation mode"
            echo "  ./start.sh --fabric                  # Connect to existing Fabric network"
            echo "  ./start.sh --deploy --fabric         # Deploy chaincode and connect"
            echo "  ./start.sh --fabric --port 8081      # Use custom port"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo ""
print_info "Configuration:"
echo "  Deploy Chaincode: $DEPLOY_CHAINCODE"
echo "  Use Fabric: $USE_FABRIC"
echo "  Port: $PORT"
echo "  Batch Size: $BATCH_SIZE"
echo "  Batch Timeout: $BATCH_TIMEOUT"
echo ""

# Step 1: Check if port is available
if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
    print_warning "Port $PORT is already in use"
    read -p "Kill the process and continue? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        lsof -ti:$PORT | xargs kill -9 2>/dev/null || true
        sleep 1
        print_success "Port $PORT cleared"
    else
        print_error "Cannot start server - port $PORT is in use"
        exit 1
    fi
fi

# Step 2: Deploy chaincode if requested
if [ "$DEPLOY_CHAINCODE" = true ]; then
    print_info "Deploying chaincode to Fabric network..."
    if [ -f "$SCRIPT_DIR/deploy-chaincode.sh" ]; then
        "$SCRIPT_DIR/deploy-chaincode.sh"
        if [ $? -eq 0 ]; then
            print_success "Chaincode deployed successfully"
        else
            print_error "Chaincode deployment failed"
            exit 1
        fi
    else
        print_error "deploy-chaincode.sh not found"
        exit 1
    fi
    echo ""
fi

# Step 3: Check Fabric network if --fabric flag is used
if [ "$USE_FABRIC" = true ]; then
    print_info "Checking Fabric network status..."
    
    # Check if Docker is running
    if ! docker ps >/dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    
    # Check if Fabric network is running
    if ! docker ps | grep -q "peer0.org1.example.com"; then
        print_warning "Fabric network is not running"
        echo ""
        echo "To start the Fabric network, run:"
        echo "  cd $PROJECT_ROOT/../fabric-samples/test-network"
        echo "  ./network.sh up createChannel"
        echo ""
        echo "Or run this script with --deploy flag to automatically deploy"
        exit 1
    fi
    
    print_success "Fabric network is running"
    
    # Check if chaincode is deployed
    print_info "Checking if chaincode is deployed..."
    FABRIC_TEST_NETWORK="$PROJECT_ROOT/../fabric-samples/test-network"
    cd "$FABRIC_TEST_NETWORK"
    export PATH=${PWD}/../bin:$PATH
    export FABRIC_CFG_PATH=$PWD/../config/
    export CORE_PEER_TLS_ENABLED=true
    export CORE_PEER_LOCALMSPID="Org1MSP"
    export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
    export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
    export CORE_PEER_ADDRESS=localhost:7051
    
    if peer lifecycle chaincode querycommitted -C mychannel -n wallet >/dev/null 2>&1; then
        print_success "Chaincode 'wallet' is deployed"
    else
        print_warning "Chaincode 'wallet' is not deployed"
        echo ""
        echo "To deploy the chaincode, run:"
        echo "  ./deploy-chaincode.sh"
        echo ""
        echo "Or run this script with --deploy flag"
        exit 1
    fi
    
    cd "$PROJECT_ROOT"
fi

# Step 4: Build the application
print_info "Building application..."
go build -o priority-gateway ./cmd/server.go >/dev/null 2>&1
if [ $? -eq 0 ]; then
    print_success "Build successful"
else
    print_error "Build failed"
    exit 1
fi

# Step 5: Start the server
echo ""
print_success "Starting Priority Fabric Transaction Gateway..."
echo ""

# Build command - only add flags if they were provided
CMD="./priority-gateway"

if [ -n "$PORT" ]; then
    CMD="$CMD --port=$PORT"
fi

if [ -n "$BATCH_SIZE" ]; then
    CMD="$CMD --batch-size=$BATCH_SIZE"
fi

if [ -n "$BATCH_TIMEOUT" ]; then
    CMD="$CMD --batch-timeout=$BATCH_TIMEOUT"
fi

if [ "$USE_FABRIC" = true ]; then
    CMD="$CMD --use-fabric"
fi

# Print startup info
echo "================================================"
echo "🌐 Server will start on: http://localhost:${PORT:-8080}"
echo "📋 Mode: $(if [ "$USE_FABRIC" = true ]; then echo "Fabric Integration"; else echo "Simulation"; fi)"
echo "================================================"
echo ""
echo "Available endpoints:"
echo "  POST   http://localhost:$PORT/submit"
echo "  GET    http://localhost:$PORT/mempool/status"
echo "  GET    http://localhost:$PORT/batcher/status"
echo "  GET    http://localhost:$PORT/transactions/completed"
echo "  GET    http://localhost:$PORT/health"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

# Start the server
exec $CMD
