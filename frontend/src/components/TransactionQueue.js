import React from "react";
import "./TransactionQueue.css";

const TransactionQueue = ({ transactions }) => {
  const getPriorityColor = (priority) => {
    switch (priority) {
      case 0:
        return "#ff4444";
      case 1:
        return "#ff8800";
      case 2:
        return "#ffcc00";
      case 3:
        return "#44ff44";
      default:
        return "#999";
    }
  };

  const getPriorityName = (priority) => {
    switch (priority) {
      case 0:
        return "Swap";
      case 1:
        return "Borrow";
      case 2:
        return "Lend";
      case 3:
        return "Transfer";
      default:
        return "Unknown";
    }
  };

  const formatTimestamp = (timestamp) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString();
  };

  return (
    <div className="transaction-queue">
      <h2>📋 Transaction Queue ({transactions.length})</h2>
      <div className="queue-container">
        {transactions.length === 0 ? (
          <div className="no-transactions">No transactions yet</div>
        ) : (
          <div className="transaction-list">
            {transactions.slice(0, 20).map((tx) => (
              <div key={tx.id} className="transaction-item">
                <div className="tx-header">
                  <span
                    className="priority-indicator"
                    style={{
                      backgroundColor: getPriorityColor(tx.priority),
                    }}
                  >
                    P{tx.priority}
                  </span>
                  <span className="tx-type">
                    {getPriorityName(tx.priority)}
                  </span>
                  <span className="tx-time">
                    {formatTimestamp(tx.timestamp)}
                  </span>
                </div>
                <div className="tx-details">
                  <div className="tx-detail">
                    <span className="label">ID:</span>
                    <span className="value">{tx.id.substring(0, 12)}...</span>
                  </div>
                  <div className="tx-detail">
                    <span className="label">From:</span>
                    <span className="value">{tx.from}</span>
                  </div>
                  <div className="tx-detail">
                    <span className="label">To:</span>
                    <span className="value">{tx.to}</span>
                  </div>
                  <div className="tx-detail">
                    <span className="label">Amount:</span>
                    <span className="value">{tx.amount}</span>
                  </div>
                </div>
                <div className="tx-status">
                  <span className={`status-badge ${tx.status}`}>
                    {tx.status}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default TransactionQueue;
