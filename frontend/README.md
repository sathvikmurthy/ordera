# Priority Fabric Transaction Gateway - Frontend Dashboard

Real-time WebSocket-powered React dashboard for monitoring the Priority Fabric Transaction Gateway system.

## Features

- **Real-Time Updates**: WebSocket connection provides live updates for all transaction and batch events
- **Transaction Submission**: Submit new transactions directly from the UI
- **Live Countdown Timer**: Shows time remaining until next batch processing
- **Priority Distribution Chart**: Visual representation of transactions by priority level
- **Transaction Queue**: Real-time view of pending transactions
- **Batch History**: Track batch processing with detailed statistics
- **System Statistics**: Monitor mempool and batcher performance metrics

## Priority Levels

- **Priority 0 (Swap)**: Highest priority - Red
- **Priority 1 (Borrow)**: High priority - Orange
- **Priority 2 (Lend)**: Medium priority - Yellow
- **Priority 3 (Transfer)**: Lowest priority - Green

## Getting Started

### Prerequisites

- Node.js (v14 or higher)
- Backend server running on `http://localhost:8080`

### Installation

```bash
cd frontend
npm install
```

### Running the Application

```bash
npm start
```

The application will open at `http://localhost:3002` and connect to the backend WebSocket at `ws://localhost:8080/ws`.

### Building for Production

```bash
npm run build
```

This creates an optimized production build in the `build/` directory.

## WebSocket Events

The dashboard listens for the following event types from the backend:

### Event Types

1. **initial_state**: Connection confirmation
2. **tx_submitted**: New transaction added to queue
3. **batch_countdown**: Countdown timer update (every second)
4. **batch_started**: Batch processing initiated
5. **batch_completed**: Batch processing finished
6. **mempool_stats**: Mempool statistics update
7. **batcher_stats**: Batcher statistics update

### Event Structure

```json
{
  "type": "event_type",
  "data": {
    /* event-specific data */
  },
  "timestamp": 1234567890
}
```

## Components

### TransactionForm

Form for submitting new transactions to the gateway.

### CountdownTimer

Displays countdown to next batch or current batch processing status.

### StatsPanel

Shows system statistics including mempool size, batch count, and total processed transactions.

### PriorityChart

Pie chart visualization of transaction distribution by priority level.

### TransactionQueue

Scrollable list of pending transactions with priority indicators.

### BatchHistory

History of processed batches with details and priority breakdowns.

## API Endpoints

The frontend communicates with these backend endpoints:

- `POST /submit` - Submit new transaction
- `GET /mempool/status` - Get mempool statistics
- `GET /batcher/status` - Get batcher status
- `GET /health` - Health check
- `WS /ws` - WebSocket connection for real-time updates

## Configuration

### Port Configuration

The frontend runs on port 3002 by default (configured in `.env` file):

```
PORT=3002
```

### WebSocket URL

The WebSocket URL can be modified in `src/App.js`:

```javascript
const WS_URL = "ws://localhost:8080/ws";
```

## Styling

The dashboard uses a modern gradient purple theme with:

- Responsive grid layouts
- Smooth animations and transitions
- Custom scrollbars
- Hover effects and visual feedback

## Technologies Used

- **React**: UI framework
- **Recharts**: Chart visualization library
- **WebSocket API**: Real-time communication
- **CSS3**: Styling with gradients, animations, and flexbox/grid layouts

## Development

### Project Structure

```
frontend/
├── public/
├── src/
│   ├── components/
│   │   ├── BatchHistory.js
│   │   ├── BatchHistory.css
│   │   ├── CountdownTimer.js
│   │   ├── CountdownTimer.css
│   │   ├── PriorityChart.js
│   │   ├── PriorityChart.css
│   │   ├── StatsPanel.js
│   │   ├── StatsPanel.css
│   │   ├── TransactionForm.js
│   │   ├── TransactionForm.css
│   │   ├── TransactionQueue.js
│   │   └── TransactionQueue.css
│   ├── hooks/
│   │   └── useWebSocket.js
│   ├── App.js
│   ├── App.css
│   └── index.js
└── package.json
```

### Available Scripts

- `npm start` - Run development server
- `npm test` - Run tests
- `npm run build` - Create production build
- `npm run eject` - Eject from Create React App (irreversible)

## Troubleshooting

### WebSocket Connection Issues

If the WebSocket fails to connect:

1. Ensure the backend server is running on port 8080
2. Check that CORS is properly configured in the backend
3. Verify the WebSocket URL in `src/App.js`
4. Check browser console for connection errors

### Display Issues

- Clear browser cache if styles don't load
- Check browser console for JavaScript errors
- Ensure all dependencies are installed (`npm install`)

## License

This project is part of the Priority Fabric Transaction Gateway system.
