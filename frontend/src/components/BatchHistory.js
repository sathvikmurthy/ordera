import React from "react";
import "./BatchHistory.css";

const BatchHistory = ({ batches, currentBatch }) => {
  const formatTime = (timestamp) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString();
  };

  return (
    <div className="batch-history">
      <h2>🔄 Batch History</h2>
      <div className="history-container">
        {currentBatch && (
          <div className="batch-item current">
            <div className="batch-number">
              Batch #{currentBatch.batchNumber}
            </div>
            <div className="batch-info-grid">
              <div className="info-item">
                <span className="info-label">Status:</span>
                <span className="info-value processing">Processing...</span>
              </div>
              <div className="info-item">
                <span className="info-label">Size:</span>
                <span className="info-value">{currentBatch.batchSize} txs</span>
              </div>
              <div className="info-item">
                <span className="info-label">Trigger:</span>
                <span className="info-value">{currentBatch.trigger}</span>
              </div>
              <div className="info-item">
                <span className="info-label">Mode:</span>
                <span className="info-value mode">{currentBatch.mode}</span>
              </div>
            </div>
            {currentBatch.priorityCount && (
              <div className="priority-counts">
                {Object.entries(currentBatch.priorityCount).map(
                  ([priority, count]) =>
                    count > 0 && (
                      <span
                        key={priority}
                        className={`priority-tag p${priority}`}
                      >
                        P{priority}: {count}
                      </span>
                    )
                )}
              </div>
            )}
          </div>
        )}

        {batches.length === 0 && !currentBatch ? (
          <div className="no-history">No batch history yet</div>
        ) : (
          batches.map((batch) => (
            <div key={batch.batchNumber} className="batch-item completed">
              <div className="batch-number">Batch #{batch.batchNumber}</div>
              <div className="batch-info-grid">
                <div className="info-item">
                  <span className="info-label">Status:</span>
                  <span className="info-value completed">✓ Completed</span>
                </div>
                <div className="info-item">
                  <span className="info-label">Size:</span>
                  <span className="info-value">{batch.batchSize} txs</span>
                </div>
                <div className="info-item">
                  <span className="info-label">Time:</span>
                  <span className="info-value">
                    {formatTime(batch.completedAt)}
                  </span>
                </div>
                <div className="info-item">
                  <span className="info-label">Total:</span>
                  <span className="info-value">{batch.totalProcessed} txs</span>
                </div>
              </div>
              {batch.priorityCount && (
                <div className="priority-counts">
                  {Object.entries(batch.priorityCount).map(
                    ([priority, count]) =>
                      count > 0 && (
                        <span
                          key={priority}
                          className={`priority-tag p${priority}`}
                        >
                          P{priority}: {count}
                        </span>
                      )
                  )}
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default BatchHistory;
