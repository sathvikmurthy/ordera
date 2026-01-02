# Priority Fabric Transaction Gateway

A priority-based transaction ordering system integrated with Hyperledger Fabric, featuring a mempool with anti-starvation mechanisms and batch processing.

## 🎯 What This System Does

This project implements a **priority-based transaction gateway** that:

1. **Accepts transactions** via HTTP API with different priority levels (swap, borrow, lend, transfer)
2. **Queues transactions** in a priority mempool with anti-starvation protection
3. **Batches transactions** for efficient processing
4. **Submits to Fabric** through the complete consensus process:
   - ✅ Endorsement by peers
   - ✅ Ordering by orderer service
   - ✅ Validation and commitment to distributed ledger
   - ✅ Consensus through Raft/BFT

## 🏗️ Architecture

```
┌─────────────┐
│   Clients   │
└──────┬──────┘
       │ HTTP POST /submit
       ▼
┌─────────────────────────────────┐
│  Transaction Gateway (server.go)│
└──────┬──────────────────────────┘
       │
       ▼
┌──────────────────┐
│  Priority Mempool │  ← Transactions queued by priority
│  - swap (0)       │
│  - borrow (1)     │
│  - lend (2)       │
│  - transfer (3)   │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│     Batcher      │  ← Groups transactions into batches
│  - Size trigger  │     (alternates between priority-only
│  - Time trigger  │      and anti-starvation modes)
└──────┬───────────┘
       │
       ▼
┌──────────────────────────────────┐
│  Fabric Gateway SDK (gRPC)       │
└──────┬───────────────────────────┘
       │
       ▼
┌──────────────────────────────────┐
│  Hyperledger Fabric Network      │
│  ┌────────┐  ┌────────┐          │
│  │ Peer 1 │  │ Peer 2 │ Endorse  │
│  └────┬───┘  └────┬───┘          │
│       │           │              │
│       └─────┬─────┘              │
│             ▼                    │
│       ┌──────────┐               │
│       │ Orderer  │  Order        │
│       └─────┬────┘               │
│             │                    │
│             ▼                    │
│      ┌──────────────┐            │
│      │ Commit Block │            │
│      │  to Ledger   │            │
│      └──────────────┘            │
└──────────────────────────────────┘
```

## 📋 Prerequisites

- **Go** 1.21 or higher
- **Docker** and **Docker Compose**
- **Hyperledger Fabric binaries** (included in fabric-samples)
- **fabric-samples** repository (already present in your setup)

## 🚀 Quick Start

### Step 1: Deploy Chaincode to Fabric Network

The deployment script will automatically:

- Start the Fabric test-network (if not running)
- Package your chaincode
- Install on both org peers
- Approve for both organizations
- Commit to the channel

```bash
cd /Users/sathvikcustiv/fabric-dev/priority-fabric-project
./deploy-chaincode.sh
```

**What happens during deployment:**

1. **Network Check**: Verifies if Fabric network is running, starts if needed
2. **Package**: Creates a chaincode package from your Go code
3. **Install**: Installs chaincode on Org1 and Org2 peers
4. **Approve**: Gets approval from both organizations
5. **Commit**: Commits chaincode definition to the channel
6. **Verify**: Confirms successful deployment

### Step 2: Start Gateway Server with Fabric Integration

```bash
# Run in simulation mode (no Fabric connection)
go run . --port=8080

# Run with Fabric integration (connects to network)
go run . --port=8080 --use-fabric

# Custom configuration
go run . --port=8080 --use-fabric --batch-size=50 --batch-timeout=5s --mempool-size=2000
```

**Command-line flags:**

- `--port`: HTTP server port (default: 8080)
- `--use-fabric`: Connect to Fabric network (default: false - simulation mode)
- `--batch-size`: Transactions per batch (default: 100)
- `--batch-timeout`: Max time before processing batch (default: 2s)
- `--mempool-size`: Maximum mempool capacity (default: 1000)
- `--verbose`: Enable verbose logging (default: false)

### Step 3: Submit Transactions

```bash
# Create wallets first (run these commands in a new terminal)
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{
    "from": "wallet1",
    "to": "wallet2",
    "amount": "100",
    "txType": "swap"
  }'
```

## 📡 API Endpoints

### POST /submit

Submit a new transaction to the mempool.

**Request:**

```json
{
  "from": "address1",
  "to": "address2",
  "amount": "100.50",
  "txType": "swap"
}
```

**Valid Transaction Types:**

- `swap` - Priority 0 (highest)
- `borrow` - Priority 1
- `lend` - Priority 2
- `transfer` - Priority 3 (lowest)

**Response:**

```json
{
  "transactionId": "a1b2c3d4...",
  "status": "queued",
  "priority": 0,
  "message": "Transaction queued with priority 0"
}
```

### GET /mempool/status

View current mempool statistics.

```bash
curl http://localhost:8080/mempool/status
```

### GET /batcher/status

View batcher statistics and processing info.

```bash
curl http://localhost:8080/batcher/status
```

### GET /transaction/status?id=<txid>

Check status of a specific transaction.

```bash
curl http://localhost:8080/transaction/status?id=a1b2c3d4
```

### GET /transactions/completed

View all completed transactions.

```bash
curl http://localhost:8080/transactions/completed
```

### GET /health

Health check endpoint.

```bash
curl http://localhost:8080/health
```

## 🔄 How Priority & Anti-Starvation Works

### Priority Levels

Transactions are ordered by priority (0 = highest, 3 = lowest):

1. **swap** (0) - Highest priority, processed first
2. **borrow** (1) - High priority
3. **lend** (2) - Medium priority
4. **transfer** (3) - Lowest priority

### Anti-Starvation Mechanism

The batcher alternates between two modes:

**Odd Batches (1, 3, 5...)**: Priority-only mode

- Strictly processes by priority
- Highest priority transactions first

**Even Batches (2, 4, 6...)**: Quota-based mode

- Each priority level gets fair quota
- Prevents low-priority transactions from being starved
- Ensures all priorities eventually get processed

## 🔗 Fabric Integration Details

### What Happens When You Submit a Transaction

1. **Client submits** via HTTP POST to gateway
2. **Gateway validates** and adds to mempool (sorted by priority)
3. **Batcher extracts** batch when size/timeout threshold reached
4. **For each transaction in batch**:
   - Transaction is sent to Fabric peer via gRPC
   - Peer **endorses** the transaction (executes chaincode)
   - Endorsement returned to gateway
5. **Gateway submits** endorsed transaction to orderer
6. **Orderer** orders transactions into a block
7. **Block is broadcast** to all peers
8. **Peers validate** and commit block to ledger
9. **Transaction confirmed** - now permanently on blockchain

### Consensus Process

Your transactions go through Fabric's complete consensus:

- **Endorsement**: Peers execute chaincode and sign results
- **Ordering**: Orderer service sequences transactions
- **Validation**: Peers verify endorsements and check for conflicts
- **Commitment**: Valid transactions written to ledger

## 🧪 Testing

### Run the included test script

```bash
# Test with simulation (no Fabric needed)
./test_anti_starvation.sh http://localhost:8080

# The script will:
# - Submit 10+ transactions with different priorities
# - Show how batching works
# - Demonstrate anti-starvation mechanism
```

### Manual testing

```bash
# Terminal 1: Start server
go run . --use-fabric --port=8080 --batch-size=5 --batch-timeout=10s

# Terminal 2: Submit various transactions
# High priority swap
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{"from":"user1","to":"user2","amount":"100","txType":"swap"}'

# Low priority transfer
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{"from":"user3","to":"user4","amount":"50","txType":"transfer"}'

# Check mempool
curl http://localhost:8080/mempool/status

# View completed transactions
curl http://localhost:8080/transactions/completed
```

## 📁 Project Structure

```
priority-fabric-project/
├── chaincode/              # Smart contract (deployed to Fabric)
│   ├── main.go            # Chaincode implementation
│   └── go.mod
├── types/                  # Shared data types
│   ├── transaction.go     # Transaction structures
│   ├── wallet.go          # Wallet structures
│   └── ...
├── fabric_client.go        # Fabric Gateway SDK client
├── batcher.go             # Transaction batching logic
├── mempool.go             # Priority mempool implementation
├── gateway.go             # HTTP API gateway
├── server.go              # Main server entry point
├── priority_queue.go      # Priority queue data structure
├── deploy-chaincode.sh    # Chaincode deployment script
└── README.md              # This file
```

## 🔍 Monitoring & Debugging

### View server logs

The server provides detailed logging of:

- Transaction submissions
- Mempool operations
- Batch processing
- Fabric network communication

### Check Fabric network

```bash
# View running containers
docker ps

# View peer logs
docker logs peer0.org1.example.com

# View orderer logs
docker logs orderer.example.com
```

### Query chaincode directly

```bash
# Set environment for Org1
cd /Users/sathvikcustiv/fabric-dev/fabric-samples/test-network
export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=$PWD/../config/
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051

# Create a wallet
peer chaincode invoke \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer.example.com \
  --tls \
  --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
  -C mychannel \
  -n wallet \
  -c '{"function":"CreateWallet","Args":[]}'

# Query a wallet (use address from CreateWallet response)
peer chaincode query \
  -C mychannel \
  -n wallet \
  -c '{"function":"GetWallet","Args":["<wallet-address>"]}'
```

## 🛠️ Troubleshooting

### "Failed to connect to Fabric"

- Ensure Fabric network is running: `cd fabric-samples/test-network && ./network.sh up createChannel`
- Check Docker containers are running: `docker ps`
- Verify chaincode is deployed: Run `./deploy-chaincode.sh`

### "Chaincode not found"

- Deploy chaincode: `./deploy-chaincode.sh`
- Check deployment: `peer lifecycle chaincode querycommitted -C mychannel -n wallet`

### Port already in use

- Kill process on port: `lsof -ti:8080 | xargs kill -9`
- Or use different port: `go run . --port=8081`

### Simulation mode when expecting Fabric

- Ensure you're using `--use-fabric` flag
- Check network connectivity to peer (localhost:7051)

## 📚 Learn More

- [Hyperledger Fabric Documentation](https://hyperledger-fabric.readthedocs.io/)
- [Fabric Gateway SDK](https://hyperledger.github.io/fabric-gateway/)
- [Understanding Fabric Transaction Flow](https://hyperledger-fabric.readthedocs.io/en/latest/txflow.html)

## 🎓 Key Concepts Explained

### Mempool

A temporary storage area where transactions wait before being processed. Like a priority queue at a bank - VIP customers (high priority) get served first, but regular customers aren't ignored forever.

### Batching

Grouping multiple transactions together for efficiency. Instead of submitting transactions one-by-one to Fabric, we batch them to reduce network overhead and improve throughput.

### Endorsement

Peers execute the chaincode and sign the result. This proves the transaction was validated by trusted parties before being added to the ledger.

### Consensus

The distributed agreement process ensuring all peers have the same ledger state. Your transactions must pass through this to be considered valid.

### Anti-Starvation

Mechanism ensuring low-priority transactions eventually get processed, even when high-priority transactions keep arriving.

---

**Built with ❤️ for Hyperledger Fabric**
