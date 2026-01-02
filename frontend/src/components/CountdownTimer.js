import React from "react";
import "./CountdownTimer.css";

const CountdownTimer = ({ countdown, currentBatch }) => {
  if (currentBatch) {
    return (
      <div className="countdown-container processing">
        <h2>⚡ Batch Processing</h2>
        <div className="batch-info">
          <div className="batch-detail">
            <span className="label">Batch #:</span>
            <span className="value">{currentBatch.batchNumber}</span>
          </div>
          <div className="batch-detail">
            <span className="label">Size:</span>
            <span className="value">{currentBatch.batchSize} txs</span>
          </div>
          <div className="batch-detail">
            <span className="label">Trigger:</span>
            <span className="value">{currentBatch.trigger}</span>
          </div>
          <div className="batch-detail">
            <span className="label">Mode:</span>
            <span className="value mode">{currentBatch.mode}</span>
          </div>
        </div>
        <div className="processing-animation">
          <div className="spinner"></div>
          <p>Processing transactions...</p>
        </div>
      </div>
    );
  }

  if (!countdown || countdown.mempoolSize === 0) {
    return (
      <div className="countdown-container idle">
        <h2>⏸️ Waiting for Transactions</h2>
        <p className="idle-message">No pending transactions in mempool</p>
      </div>
    );
  }

  const progress =
    (countdown.remainingSeconds /
      (countdown.batchSize > 0 ? 30 : countdown.remainingSeconds)) *
    100;

  return (
    <div className="countdown-container active">
      <h2>⏱️ Next Batch In</h2>
      <div className="countdown-display">
        <div className="time-remaining">
          <span className="seconds">{countdown.remainingSeconds}</span>
          <span className="unit">seconds</span>
        </div>
        <div className="progress-bar">
          <div
            className="progress-fill"
            style={{ width: `${100 - progress}%` }}
          ></div>
        </div>
      </div>
      <div className="countdown-stats">
        <div className="stat">
          <span className="stat-label">Queued:</span>
          <span className="stat-value">{countdown.mempoolSize} txs</span>
        </div>
        <div className="stat">
          <span className="stat-label">Batch Size:</span>
          <span className="stat-value">{countdown.batchSize} txs</span>
        </div>
        <div className="stat">
          <span className="stat-label">Need:</span>
          <span className="stat-value">
            {Math.max(0, countdown.batchSize - countdown.mempoolSize)} more
          </span>
        </div>
      </div>
    </div>
  );
};

export default CountdownTimer;
