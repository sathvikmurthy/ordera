# 🚀 Complete Startup Guide - Priority Fabric Transaction Gateway

This guide provides step-by-step instructions for running your Hyperledger Fabric project with the priority-based transaction gateway and batcher.

## 📊 Current Status

✅ **Fabric Network**: Running (31 minutes uptime)

- Orderer: `orderer.example.com` (Port 7050)
- Peer: `peer0.org1.example.com` (Port 7051)

❌ **Chaincode**: Not deployed yet (namespace 'wallet' not defined)

## 🎯 Complete Startup Procedure

### Step 1: Deploy the Chaincode

The chaincode must be deployed before you can submit transactions. Run the deployment script:

```bash
cd /Users/sathvikcustiv/fabric-dev/priority-fabric-project
./deploy-chaincode.sh
```

**What this does:**

- Verifies Fabric network is running
- Packages the wallet chaincode
- Installs on Org1 and Org2 peers
- Approves for both organizations
- Commits to the mychannel channel
- Verifies deployment

**Expected output:**

```
🎉 SUCCESS! Chaincode deployed successfully!

📋 Deployment Summary:
   Chaincode Name: wallet
   Version: 1.5
   Channel: mychannel
   Package ID: wallet_1.5:xxxxx
```

### Step 2: Start the Gateway Server with Batcher

Once chaincode is deployed, start the gateway server that includes the batcher:

```bash
# Option A: Using the convenience script
./start.sh --fabric --port 8080

# Option B: Direct Go execution
go run . --port=8080 --use-fabric --batch-size=100 --batch-timeout=2s
```

**Command-line flags:**

- `--port=8080` - HTTP server port
- `--use-fabric` - Connect to Fabric network (REQUIRED for real transactions)
- `--batch-size=100` - Number of transactions per batch
- `--batch-timeout=2s` - Maximum time before processing batch
- `--mempool-size=1000` - Maximum mempool capacity
- `--verbose` - Enable detailed logging

**What the batcher does:**

- Collects transactions from the mempool
- Groups them into batches based on size/timeout triggers
- Alternates between priority-only and anti-starvation modes
- Submits batches to Fabric for consensus and commitment

**Server will start on:** `http://localhost:8080`

### Step 3: Submit Transactions via curl

Now you can submit transactions through the HTTP API. The gateway will queue them in the mempool, and the batcher will process them.

#### Example 1: Submit a high-priority SWAP transaction

```bash
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{
    "from": "wallet1",
    "to": "wallet2",
    "amount": "100",
    "txType": "swap"
  }'
```

**Response:**

```json
{
  "transactionId": "abc123...",
  "status": "queued",
  "priority": 0,
  "message": "Transaction queued with priority 0"
}
```

#### Example 2: Submit a BORROW transaction

```bash
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{
    "from": "wallet3",
    "to": "wallet4",
    "amount": "500",
    "txType": "borrow"
  }'
```

#### Example 3: Submit a TRANSFER transaction (lowest priority)

```bash
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{
    "from": "wallet5",
    "to": "wallet6",
    "amount": "50",
    "txType": "transfer"
  }'
```

#### Priority Levels:

- `swap` - Priority 0 (highest) - Processed first
- `borrow` - Priority 1 (high)
- `lend` - Priority 2 (medium)
- `transfer` - Priority 3 (lowest)

### Step 4: Monitor Transaction Processing

#### Check Mempool Status

```bash
curl http://localhost:8080/mempool/status
```

**Response shows:**

```json
{
  "totalTransactions": 15,
  "byPriority": {
    "0": 5, // swap
    "1": 3, // borrow
    "2": 4, // lend
    "3": 3 // transfer
  },
  "oldestTransaction": "2024-11-27T12:20:00Z"
}
```

#### Check Batcher Status

```bash
curl http://localhost:8080/batcher/status
```

**Response shows:**

```json
{
  "currentBatchNumber": 5,
  "mode": "anti-starvation",
  "transactionsProcessed": 450,
  "batchesCompleted": 4,
  "averageBatchSize": 112.5
}
```

#### View Completed Transactions

```bash
curl http://localhost:8080/transactions/completed
```

#### Check Specific Transaction Status

```bash
curl http://localhost:8080/transaction/status?id=abc123
```

## 🔄 How the Complete Flow Works

```
1. Client submits transaction via curl
   ↓
2. Gateway receives and validates transaction
   ↓
3. Transaction added to Priority Mempool (sorted by priority)
   ↓
4. Batcher monitors mempool for batch triggers:
   - Size threshold (e.g., 100 transactions)
   - Time threshold (e.g., 2 seconds)
   ↓
5. Batcher extracts batch (alternates priority/anti-starvation modes)
   ↓
6. For each transaction in batch:
   - Sent to Fabric peer via gRPC
   - Peer endorses (executes chaincode)
   - Endorsement returned to gateway
   ↓
7. Gateway submits endorsed transactions to orderer
   ↓
8. Orderer creates block and broadcasts to peers
   ↓
9. Peers validate and commit block to ledger
   ↓
10. Transaction confirmed ✅ (permanently on blockchain)
```

## 📝 Testing the Anti-Starvation Mechanism

Use the provided test script to see how the batcher handles priority and anti-starvation:

```bash
cd /Users/sathvikcustiv/fabric-dev/priority-fabric-project
./test_anti_starvation.sh http://localhost:8080
```

**What the test does:**

- Submits 10+ transactions with mixed priorities
- Shows batch 1 (priority-only): High-priority transactions processed first
- Shows batch 2 (anti-starvation): Fair quota for all priority levels
- Demonstrates that low-priority transactions aren't starved

## 🛠️ Troubleshooting

### Issue: "namespace wallet is not defined"

**Solution:** Chaincode not deployed. Run:

```bash
./deploy-chaincode.sh
```

### Issue: "Failed to connect to Fabric"

**Solution:** Check network status:

```bash
docker ps | grep peer0.org1.example.com
```

If not running:

```bash
cd /Users/sathvikcustiv/fabric-dev/fabric-samples/test-network
./network.sh up createChannel
```

### Issue: Port 8080 already in use

**Solution:** Kill existing process or use different port:

```bash
# Option 1: Kill process on port 8080
lsof -ti:8080 | xargs kill -9

# Option 2: Use different port
go run . --port=8081 --use-fabric
```

### Issue: Batcher not processing transactions

**Check:**

1. Batch size threshold: Have enough transactions been queued?
2. Batch timeout: Has enough time passed?
3. Server logs: Look for error messages
4. Fabric connection: Ensure `--use-fabric` flag is set

### Issue: Transactions stuck in mempool

**Solution:** Check batcher logs and verify:

- Fabric network is running
- Chaincode is deployed
- No endorsement failures in logs

## 📊 Monitoring Commands

### View all Docker containers

```bash
docker ps
```

### View peer logs

```bash
docker logs peer0.org1.example.com -f
```

### View orderer logs

```bash
docker logs orderer.example.com -f
```

### Check chaincode deployment

```bash
cd /Users/sathvikcustiv/fabric-dev/fabric-samples/test-network
export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=$PWD/../config/
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051

peer lifecycle chaincode querycommitted -C mychannel -n wallet
```

## 🎯 Quick Reference Commands

### Full Startup Sequence

```bash
# Terminal 1: Deploy chaincode (one-time setup)
cd /Users/sathvikcustiv/fabric-dev/priority-fabric-project
./deploy-chaincode.sh

# Terminal 1: Start gateway with batcher
./start.sh --fabric --port 8080

# Terminal 2: Submit transactions
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{"from":"wallet1","to":"wallet2","amount":"100","txType":"swap"}'

# Terminal 2: Monitor status
curl http://localhost:8080/mempool/status
curl http://localhost:8080/batcher/status
curl http://localhost:8080/transactions/completed
```

## 📖 Additional Resources

- **Full README**: `priority-fabric-project/README.md`
- **Dashboard Guide**: `priority-fabric-project/DASHBOARD_README.md`
- **Test Script**: `priority-fabric-project/test_anti_starvation.sh`
- **Fabric Docs**: https://hyperledger-fabric.readthedocs.io/

## ✅ Summary

**To run your project:**

1. **Deploy chaincode** (one-time): `./deploy-chaincode.sh`
2. **Start server with batcher**: `./start.sh --fabric --port 8080`
3. **Submit transactions**: Use curl with POST to `/submit` endpoint
4. **Monitor**: Check `/mempool/status`, `/batcher/status`, `/transactions/completed`
