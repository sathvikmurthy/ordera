package types

// MempoolStats provides statistics about the mempool
type MempoolStats struct {
    TotalTransactions int            `json:"totalTransactions"`
    MaxSize           int            `json:"maxSize"`
    ByPriority        map[int]int    `json:"byPriority"` // priority -> count
    PendingTxs        []*Transaction `json:"pendingTransactions"`
}

// BatcherStats provides statistics about the batcher
type BatcherStats struct {
    Running      bool   `json:"running"`
    BatchSize    int    `json:"batchSize"`
    BatchTimeout string `json:"batchTimeout"`
    MempoolSize  int    `json:"mempoolSize"`
}