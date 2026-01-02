import React from "react";
import "./StatsPanel.css";

const StatsPanel = ({ mempoolStats, batcherStats }) => {
  const priorityLabels = {
    0: "Swap",
    1: "Borrow",
    2: "Lend",
    3: "Transfer",
  };

  return (
    <div className="stats-panel">
      <h2>📊 System Statistics</h2>

      <div className="stats-grid">
        <div className="stat-card">
          <h3>Mempool</h3>
          <div className="stat-value">{mempoolStats.totalPending || 0}</div>
          <div className="stat-label">Total Pending</div>
        </div>

        <div className="stat-card">
          <h3>Capacity</h3>
          <div className="stat-value">{mempoolStats.maxSize || 0}</div>
          <div className="stat-label">Max Size</div>
        </div>

        <div className="stat-card">
          <h3>Batches</h3>
          <div className="stat-value">{batcherStats.batchesProcessed || 0}</div>
          <div className="stat-label">Processed</div>
        </div>

        <div className="stat-card">
          <h3>Total TXs</h3>
          <div className="stat-value">
            {batcherStats.totalTransactionsProcessed || 0}
          </div>
          <div className="stat-label">Completed</div>
        </div>
      </div>

      {mempoolStats.byPriority && (
        <div className="priority-breakdown">
          <h3>Queue by Priority</h3>
          <div className="priority-list">
            {Object.entries(mempoolStats.byPriority).map(
              ([priority, count]) => (
                <div key={priority} className="priority-item">
                  <span className={`priority-badge priority-${priority}`}>
                    {priority}
                  </span>
                  <span className="priority-name">
                    {priorityLabels[priority]}
                  </span>
                  <span className="priority-count">{count} txs</span>
                </div>
              )
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default StatsPanel;
