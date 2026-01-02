import React, { useState, useEffect } from "react";
import useWebSocket from "./hooks/useWebSocket";
import "./App.css";
import TransactionQueue from "./components/TransactionQueue";
import CountdownTimer from "./components/CountdownTimer";
import StatsPanel from "./components/StatsPanel";
import PriorityChart from "./components/PriorityChart";
import BatchHistory from "./components/BatchHistory";
import TransactionForm from "./components/TransactionForm";

const WS_URL = "ws://localhost:8080/ws";

function App() {
  const { isConnected, lastMessage, connectionError } = useWebSocket(WS_URL);
  const [transactions, setTransactions] = useState([]);
  const [mempoolStats, setMempoolStats] = useState({});
  const [batcherStats, setBatcherStats] = useState({});
  const [countdown, setCountdown] = useState(null);
  const [batchHistory, setBatchHistory] = useState([]);
  const [currentBatch, setCurrentBatch] = useState(null);

  useEffect(() => {
    if (!lastMessage) return;

    const { type, data } = lastMessage;

    switch (type) {
      case "initial_state":
        console.log("Connected:", data.message);
        break;

      case "tx_submitted":
        setTransactions((prev) => [
          {
            id: data.transactionId,
            from: data.from,
            to: data.to,
            amount: data.amount,
            txType: data.txType,
            priority: data.priority,
            status: data.status,
            timestamp: Date.now(),
          },
          ...prev,
        ]);
        break;

      case "batch_countdown":
        setCountdown({
          remainingSeconds: data.remainingSeconds,
          mempoolSize: data.mempoolSize,
          batchSize: data.batchSize,
        });
        setMempoolStats(data.mempoolStats || {});
        break;

      case "batch_started":
        setCurrentBatch({
          batchNumber: data.batchNumber,
          batchSize: data.batchSize,
          trigger: data.trigger,
          mode: data.mode,
          priorityCount: data.priorityCount,
          status: "processing",
        });
        setCountdown(null);
        break;

      case "batch_completed":
        if (currentBatch) {
          setBatchHistory((prev) => [
            {
              ...currentBatch,
              totalProcessed: data.totalProcessed,
              status: "completed",
              completedAt: Date.now(),
            },
            ...prev.slice(0, 9), // Keep last 10 batches
          ]);
        }
        setCurrentBatch(null);
        break;

      case "mempool_stats":
        setMempoolStats(data);
        break;

      case "batcher_stats":
        setBatcherStats(data);
        break;

      default:
        console.log("Unknown event type:", type);
    }
  }, [lastMessage, currentBatch]);

  return (
    <div className="App">
      <header className="App-header">
        <h1>🚀 Priority Fabric Transaction Gateway</h1>
        <div className="connection-status">
          {isConnected ? (
            <span className="connected">🟢 Connected</span>
          ) : (
            <span className="disconnected">
              🔴 Disconnected {connectionError && `(${connectionError})`}
            </span>
          )}
        </div>
      </header>

      <div className="dashboard">
        <div className="top-section">
          <TransactionForm />
          <CountdownTimer countdown={countdown} currentBatch={currentBatch} />
        </div>

        <div className="middle-section">
          <StatsPanel mempoolStats={mempoolStats} batcherStats={batcherStats} />
          <PriorityChart mempoolStats={mempoolStats} />
        </div>

        <div className="bottom-section">
          <TransactionQueue transactions={transactions} />
          <BatchHistory batches={batchHistory} currentBatch={currentBatch} />
        </div>
      </div>
    </div>
  );
}

export default App;
