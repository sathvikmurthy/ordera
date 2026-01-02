# Real-Time WebSocket Dashboard Setup Guide

This guide explains how to run the complete Priority Fabric Transaction Gateway with the real-time React dashboard.

## Architecture Overview

The system consists of two main components:

1. **Backend (Go)**: Priority-based transaction gateway with WebSocket support
2. **Frontend (React)**: Real-time dashboard with live updates

## Quick Start

### 1. Start the Backend Server

```bash
cd priority-fabric-project
go run *.go
```

The backend will start on `http://localhost:8080` with:

- HTTP API endpoints for transaction submission
- WebSocket endpoint at `ws://localhost:8080/ws`
- Default settings: batch-size=5, batch-timeout=30s

### 2. Start the Frontend Dashboard

In a new terminal:

```bash
cd priority-fabric-project/frontend
npm install  # First time only
npm start
```

The dashboard will open automatically at `http://localhost:3002`

## Testing the Dashboard

### Submit Transactions via UI

1. Use the "Submit Transaction" form on the dashboard
2. Fill in:
   - From: user1
   - To: user2
   - Amount: 100
   - Type: Choose priority (swap, borrow, lend, transfer)
3. Click "Submit Transaction"

### Submit Transactions via API

```bash
# Swap (Priority 0 - Highest)
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{"from":"user1","to":"user2","amount":"100","txType":"swap"}'

# Borrow (Priority 1)
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{"from":"user3","to":"user4","amount":"50","txType":"borrow"}'

# Lend (Priority 2)
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{"from":"user5","to":"user6","amount":"75","txType":"lend"}'

# Transfer (Priority 3 - Lowest)
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{"from":"user7","to":"user8","amount":"200","txType":"transfer"}'
```

## Dashboard Features

### Real-Time Updates

The dashboard displays live updates for:

- **Transaction Submission**: New transactions appear immediately in the queue
- **Countdown Timer**: Updates every second showing time until next batch
- **Batch Processing**: Visual feedback when batches are being processed
- **Priority Distribution**: Live pie chart of transaction priorities
- **System Statistics**: Real-time mempool and batcher stats

### Visual Indicators

- **Priority Colors**:

  - Red (P0): Swap - Highest priority
  - Orange (P1): Borrow
  - Yellow (P2): Lend
  - Green (P3): Transfer - Lowest priority

- **Connection Status**: Green (Connected) / Red (Disconnected)

- **Batch Mode Indicator**: Shows STANDARD or QUOTA-BASED anti-starvation mode

## Batch Processing Triggers

Batches process when either condition is met:

1. **Size-based**: 5 transactions accumulated (configurable)
2. **Time-based**: 30 seconds elapsed (configurable)

Watch the countdown timer to see when the next batch will process!

## Advanced Configuration

### Backend Configuration

Customize batch settings:

```bash
go run *.go -batch-size=10 -batch-timeout=10s -port=8080
```

Options:

- `-batch-size`: Number of transactions per batch (default: 5)
- `-batch-timeout`: Maximum wait time (default: 30s)
- `-port`: Server port (default: 8080)
- `-mempool-size`: Max mempool capacity (default: 1000)
- `-use-fabric`: Connect to Fabric network (requires network running)

### Frontend Configuration

The frontend runs on port 3002 (configured in `frontend/.env`).

To change the port, edit `frontend/.env`:

```
PORT=3002
```

To change the WebSocket URL, edit `frontend/src/App.js`:

```javascript
const WS_URL = "ws://localhost:8080/ws";
```

## API Endpoints

### HTTP Endpoints

- `POST /submit` - Submit new transaction
- `GET /mempool/status` - View mempool statistics
- `GET /batcher/status` - View batcher status
- `GET /transaction/status?id=<txid>` - Check specific transaction
- `GET /transactions/completed` - View all completed transactions
- `GET /health` - Health check

### WebSocket Endpoint

- `WS /ws` - Real-time event stream

## WebSocket Events

The frontend receives these events:

1. `tx_submitted` - New transaction added
2. `batch_countdown` - Countdown update (every second)
3. `batch_started` - Batch processing begins
4. `batch_completed` - Batch processing complete
5. `mempool_stats` - Mempool statistics
6. `batcher_stats` - Batcher statistics

## Testing Anti-Starvation

The system alternates between two batch selection modes:

- **Odd batches (1, 3, 5...)**: Standard priority ordering
- **Even batches (2, 4, 6...)**: Quota-based anti-starvation

To test:

1. Submit multiple low-priority transactions (transfer)
2. Submit high-priority transactions (swap)
3. Watch the batch history to see both modes in action
4. Verify low-priority transactions get processed in quota-based batches

## Troubleshooting

### Backend Issues

**Port already in use:**

```bash
# Find process using port 8080
lsof -i :8080
# Kill it or use different port
go run *.go -port=8081
```

**WebSocket not working:**

- Check firewall settings
- Verify CORS configuration
- Check browser console for errors

### Frontend Issues

**WebSocket connection failed:**

- Ensure backend is running on port 8080
- Check WebSocket URL in App.js
- Verify no proxy blocking WebSocket

**UI not updating:**

- Check browser console for errors
- Refresh the page
- Clear browser cache

**npm install fails:**

```bash
rm -rf node_modules package-lock.json
npm install
```

## Development Mode vs Production

### Development (Current Setup)

- Backend: `go run *.go`
- Frontend: `npm start` (development server)
- Hot reload enabled
- Debug logging active

### Production Build

Backend:

```bash
go build -o priority-gateway
./priority-gateway
```

Frontend:

```bash
cd frontend
npm run build
# Serve the build/ directory with nginx or similar
```

## Performance Tips

1. **Batch Size**: Smaller batches = faster processing but more overhead
2. **Batch Timeout**: Shorter timeout = more frequent batches
3. **Mempool Size**: Increase if handling high transaction volumes
4. **WebSocket**: Single connection handles all real-time updates efficiently

## Next Steps

1. **With Fabric**: Use `-use-fabric` flag to connect to Hyperledger Fabric
2. **Load Testing**: Use the provided test scripts to stress-test the system
3. **Monitoring**: Add Prometheus/Grafana for production monitoring
4. **Security**: Add authentication and rate limiting for production

## Support

For issues or questions:

1. Check browser console for frontend errors
2. Check terminal logs for backend errors
3. Review the README files in both backend and frontend directories

## License

Part of the Priority Fabric Transaction Gateway system.
