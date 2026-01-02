# Priority Fabric Project - Setup Guide

This guide will help you set up the Priority Fabric Transaction Gateway project from scratch on a new machine.

## 📋 Prerequisites

Before you begin, ensure you have the following installed:

### Required Software

1. **Docker** (v20.10 or higher)

   - [Install Docker Desktop](https://www.docker.com/products/docker-desktop)
   - Verify: `docker --version`

2. **Docker Compose** (v2.0 or higher)

   - Usually included with Docker Desktop
   - Verify: `docker compose version`

3. **Go** (v1.21 or higher)

   - [Install Go](https://golang.org/doc/install)
   - Verify: `go version`

4. **Node.js** (v16 or higher) and **npm**

   - [Install Node.js](https://nodejs.org/)
   - Verify: `node --version` and `npm --version`

5. **Git**

   - [Install Git](https://git-scm.com/downloads)
   - Verify: `git --version`

6. **curl** or **wget**
   - Usually pre-installed on macOS/Linux
   - Verify: `curl --version`

## 🚀 Installation Steps

### Step 1: Install Hyperledger Fabric

Clone the official Fabric samples repository and install binaries:

```bash
# Navigate to your development directory
cd ~
mkdir -p fabric-dev
cd fabric-dev

# Clone fabric-samples (includes test-network and tools)
curl -sSLO https://raw.githubusercontent.com/hyperledger/fabric/main/scripts/install-fabric.sh && chmod +x install-fabric.sh

# Download Fabric binaries, Docker images, and samples
./install-fabric.sh binary docker samples

# This will download:
# - Hyperledger Fabric Docker images (peer, orderer, CA, etc.)
# - CLI tools (peer, orderer, configtxgen, etc.)
# - fabric-samples repository with example networks
```

**Expected directory structure after this step:**

```
~/fabric-dev/
├── fabric-samples/
│   ├── bin/              # Fabric CLI binaries
│   ├── config/           # Configuration files
│   ├── test-network/     # Test network scripts
│   └── ...
```

### Step 2: Verify Fabric Installation

```bash
cd fabric-samples/test-network

# Start the test network
./network.sh up createChannel

# Verify containers are running
docker ps

# You should see containers for:
# - peer0.org1.example.com
# - peer0.org2.example.com
# - orderer.example.com
# - (and possibly CA containers)

# Bring down the network (we'll restart it later)
./network.sh down
```

### Step 3: Clone This Project

```bash
# Navigate back to fabric-dev directory
cd ~/fabric-dev

# Clone the priority-fabric-project
git clone <your-repo-url> priority-fabric-project
# OR if you have the project locally:
# cp -r /path/to/priority-fabric-project ./priority-fabric-project

cd priority-fabric-project
```

### Step 4: Install Go Dependencies

```bash
# Install main project dependencies
go mod download

# Install chaincode dependencies
cd chaincode
go mod download
cd ..

# Install types package dependencies
cd types
go mod download
cd ..
```

### Step 5: Install Frontend Dependencies

```bash
cd frontend
npm install
cd ..
```

### Step 6: Set Up Fabric Network

```bash
# Navigate to test-network
cd ../fabric-samples/test-network

# Start the Fabric network with a channel
./network.sh up createChannel -c mychannel -ca

# This will:
# - Start Docker containers for peers, orderer, and CAs
# - Create a channel named "mychannel"
# - Join both org peers to the channel
```

### Step 7: Deploy Chaincode

```bash
# Navigate back to priority-fabric-project
cd ~/fabric-dev/priority-fabric-project

# Deploy the chaincode to Fabric network
./deploy-chaincode.sh

# This script will:
# - Package the chaincode
# - Install on both organization peers
# - Approve the chaincode for both orgs
# - Commit the chaincode definition
# - Verify deployment
```

**Expected output:**

```
✓ Chaincode packaged
✓ Chaincode installed on Org1 peer
✓ Chaincode installed on Org2 peer
✓ Chaincode approved by Org1
✓ Chaincode approved by Org2
✓ Chaincode committed to channel
✓ Chaincode initialization verified
```

## 🎯 Running the Project

### Option 1: Backend Only (Go Server)

```bash
cd ~/fabric-dev/priority-fabric-project

# Run in simulation mode (no Fabric connection - for testing)
go run . --port=8080

# Run with Fabric integration (recommended)
go run . --port=8080 --use-fabric

# Run with custom configuration
go run . --port=8080 --use-fabric --batch-size=50 --batch-timeout=5s
```

### Option 2: Frontend Dashboard (Recommended)

```bash
# Terminal 1: Start Go backend server
cd ~/fabric-dev/priority-fabric-project
go run . --port=8080 --use-fabric

# Terminal 2: Start React frontend
cd ~/fabric-dev/priority-fabric-project/frontend
npm start

# Frontend will open at http://localhost:3002
# Backend API runs at http://localhost:8080
```

### Option 3: Using the Start Script

```bash
cd ~/fabric-dev/priority-fabric-project

# Make script executable (if needed)
chmod +x start.sh

# Start both backend and frontend
./start.sh
```

## ✅ Verify Everything Works

### Test 1: Check Backend Health

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Test 2: Submit a Test Transaction

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

Expected response:

```json
{
  "transactionId": "abc123...",
  "status": "queued",
  "priority": 0,
  "message": "Transaction queued with priority 0"
}
```

### Test 3: Check Mempool Status

```bash
curl http://localhost:8080/mempool/status
```

### Test 4: Run Anti-Starvation Test

```bash
cd ~/fabric-dev/priority-fabric-project
./test_anti_starvation.sh http://localhost:8080
```

## 📂 Project Directory Structure

After setup, your directory structure should look like:

```
~/fabric-dev/
├── fabric-samples/           # Official Fabric samples (boilerplate)
│   ├── bin/                 # Fabric CLI tools
│   ├── config/              # Fabric configs
│   ├── test-network/        # Test network scripts
│   └── ...
│
└── priority-fabric-project/  # Your custom project (THIS REPO)
    ├── chaincode/           # Smart contract code
    ├── frontend/            # React dashboard
    ├── types/               # Shared type definitions
    ├── *.go                 # Go source files
    ├── *.sh                 # Helper scripts
    ├── .gitignore           # Git ignore rules
    ├── README.md            # Project documentation
    └── SETUP.md             # This file
```

## 🔧 Configuration

### Environment Variables

Create a `.env` file in the project root if you need custom configuration:

```bash
# .env (optional)
FABRIC_NETWORK_PATH=../fabric-samples/test-network
FABRIC_CHANNEL=mychannel
FABRIC_CHAINCODE=wallet
SERVER_PORT=8080
BATCH_SIZE=100
BATCH_TIMEOUT=2s
```

### Frontend Configuration

The frontend `.env` file is already configured:

```bash
# frontend/.env
PORT=3002
```

Modify if needed for your setup.

## 🐛 Troubleshooting

### Issue: "Cannot connect to Docker daemon"

**Solution:**

```bash
# Make sure Docker Desktop is running
open -a Docker  # macOS
# or start Docker Desktop manually
```

### Issue: "Fabric network not found"

**Solution:**

```bash
cd ~/fabric-dev/fabric-samples/test-network
./network.sh up createChannel -c mychannel
```

### Issue: "Chaincode not installed"

**Solution:**

```bash
cd ~/fabric-dev/priority-fabric-project
./deploy-chaincode.sh
```

### Issue: "Port 8080 already in use"

**Solution:**

```bash
# Find and kill process using the port
lsof -ti:8080 | xargs kill -9

# Or use a different port
go run . --port=8081 --use-fabric
```

### Issue: "Cannot find peer command"

**Solution:**

```bash
# Add Fabric binaries to PATH
export PATH=~/fabric-dev/fabric-samples/bin:$PATH
export FABRIC_CFG_PATH=~/fabric-dev/fabric-samples/config/

# Add to ~/.zshrc or ~/.bashrc to make permanent
echo 'export PATH=$HOME/fabric-dev/fabric-samples/bin:$PATH' >> ~/.zshrc
echo 'export FABRIC_CFG_PATH=$HOME/fabric-dev/fabric-samples/config/' >> ~/.zshrc
```

### Issue: "Go module errors"

**Solution:**

```bash
# Clean and reinstall dependencies
cd ~/fabric-dev/priority-fabric-project
go clean -modcache
go mod download
go mod tidy
```

### Issue: "Frontend won't start"

**Solution:**

```bash
cd ~/fabric-dev/priority-fabric-project/frontend
rm -rf node_modules package-lock.json
npm install
npm start
```

## 🧹 Clean Up

### Stop Everything

```bash
# Stop Go server (Ctrl+C in terminal)

# Stop frontend (Ctrl+C in terminal)

# Stop Fabric network
cd ~/fabric-dev/fabric-samples/test-network
./network.sh down

# Remove Docker volumes (complete cleanup)
docker volume prune -f
```

### Remove Generated Artifacts

```bash
cd ~/fabric-dev/priority-fabric-project

# Remove compiled binaries
rm -f priority-gateway chaincode/chaincode

# Remove frontend build
rm -rf frontend/build frontend/node_modules
```

## 📚 Next Steps

1. **Read the main README.md** for detailed API documentation
2. **Check DASHBOARD_README.md** for frontend usage guide
3. **Review the code** in `server.go`, `mempool.go`, and `batcher.go`
4. **Experiment with transactions** using the test scripts
5. **Monitor logs** to understand the flow

## 🆘 Getting Help

- **Hyperledger Fabric Docs**: https://hyperledger-fabric.readthedocs.io/
- **Fabric Samples Repo**: https://github.com/hyperledger/fabric-samples
- **Fabric Gateway SDK**: https://hyperledger.github.io/fabric-gateway/

## 📝 Summary Checklist

After completing setup, verify you have:

- [ ] Docker and Docker Compose installed
- [ ] Go 1.21+ installed
- [ ] Node.js and npm installed
- [ ] fabric-samples repository cloned
- [ ] Fabric binaries and Docker images downloaded
- [ ] Test network running (`docker ps` shows peer and orderer containers)
- [ ] This project cloned/copied to `~/fabric-dev/priority-fabric-project`
- [ ] Go dependencies installed (`go mod download`)
- [ ] Frontend dependencies installed (`npm install`)
- [ ] Chaincode deployed successfully
- [ ] Backend server running on port 8080
- [ ] Frontend dashboard accessible at http://localhost:3002
- [ ] Test transaction submitted successfully

**🎉 You're all set! Happy coding!**
