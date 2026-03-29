package main

import (
    "container/heap"
    "fmt"
    "sync"
    
    "priority-fabric-project/types"
)

// Mempool manages pending transactions with priority ordering
type Mempool struct {
    transactions *PriorityQueue
    mutex        sync.RWMutex
    maxSize      int
    txMap        map[string]*types.Transaction // For quick lookups by ID
}

// NewMempool creates a new mempool instance
func NewMempool(maxSize int) *Mempool {
    pq := &PriorityQueue{}
    heap.Init(pq)
    
    return &Mempool{
        transactions: pq,
        maxSize:      maxSize,
        txMap:        make(map[string]*types.Transaction),
    }
}

// AddTransaction adds a new transaction to the mempool
func (m *Mempool) AddTransaction(tx *types.Transaction) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    // Check if mempool is full
    if len(m.txMap) >= m.maxSize {
        return fmt.Errorf("mempool is full (max: %d)", m.maxSize)
    }
    
    // Check if transaction already exists
    if _, exists := m.txMap[tx.ID]; exists {
        return fmt.Errorf("transaction %s already exists in mempool", tx.ID)
    }
    
    // Set status to pending if not already set
    if tx.Status == "" {
        tx.Status = "pending"
    }
    
    // Add to priority queue and map
    heap.Push(m.transactions, tx)
    m.txMap[tx.ID] = tx
    
    return nil
}

// GetBatch extracts a batch of transactions using alternating strategies
// useQuotaMode determines the batching strategy:
// - false: Pure priority ordering (highest priority first)
// - true: Quota-based anti-starvation (at least 1 per priority)
func (m *Mempool) GetBatch(batchSize int, useQuotaMode bool) []*types.Transaction {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    if m.transactions.Len() == 0 {
        return []*types.Transaction{}
    }
    
    if !useQuotaMode {
        // Standard priority-based batching (ascending priority order)
        return m.getBatchStandard(batchSize)
    }
    
    // Quota-based anti-starvation batching
    return m.getBatchQuotaBased(batchSize)
}

// getBatchStandard extracts batch using pure priority ordering
func (m *Mempool) getBatchStandard(batchSize int) []*types.Transaction {
    var batch []*types.Transaction
    
    // Extract up to batchSize transactions in priority order
    for i := 0; i < batchSize && m.transactions.Len() > 0; i++ {
        tx := heap.Pop(m.transactions).(*types.Transaction)
        tx.SetProcessing()
        batch = append(batch, tx)
        
        // Remove from map
        delete(m.txMap, tx.ID)
    }
    
    return batch
}

// getBatchQuotaBased extracts batch with quota-based anti-starvation
// Algorithm:
// 1. Reserve at least 1 slot per priority level (if transactions exist)
// 2. Fill remaining slots with highest priority transactions
func (m *Mempool) getBatchQuotaBased(batchSize int) []*types.Transaction {
    // Group transactions by priority
    priorityGroups := make(map[int][]*types.Transaction)
    
    // Extract all transactions from heap to organize by priority
    for m.transactions.Len() > 0 {
        tx := heap.Pop(m.transactions).(*types.Transaction)
        priorityGroups[tx.Priority] = append(priorityGroups[tx.Priority], tx)
    }
    
    // Build batch with quota-based selection
    var batch []*types.Transaction
    slotsRemaining := batchSize
    
    // Phase 1: Reserve 1 slot per priority level (anti-starvation)
    // Process priorities in order: 0, 1, 2, 3
    for priority := 0; priority <= 3 && slotsRemaining > 0; priority++ {
        if len(priorityGroups[priority]) > 0 {
            // Take one transaction from this priority
            tx := priorityGroups[priority][0]
            priorityGroups[priority] = priorityGroups[priority][1:]
            
            tx.SetProcessing()
            batch = append(batch, tx)
            delete(m.txMap, tx.ID)
            slotsRemaining--
        }
    }
    
    // Phase 2: Fill remaining slots with highest priority transactions
    for priority := 0; priority <= 3 && slotsRemaining > 0; priority++ {
        for len(priorityGroups[priority]) > 0 && slotsRemaining > 0 {
            tx := priorityGroups[priority][0]
            priorityGroups[priority] = priorityGroups[priority][1:]
            
            tx.SetProcessing()
            batch = append(batch, tx)
            delete(m.txMap, tx.ID)
            slotsRemaining--
        }
    }
    
    // Put remaining transactions back into the heap
    for priority := 0; priority <= 3; priority++ {
        for _, tx := range priorityGroups[priority] {
            heap.Push(m.transactions, tx)
        }
    }
    
    return batch
}

// GetStats returns current mempool statistics
func (m *Mempool) GetStats() types.MempoolStats {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    stats := types.MempoolStats{
        TotalTransactions: len(m.txMap),
        MaxSize:           m.maxSize,
        ByPriority:        make(map[int]int),
        PendingTxs:        make([]*types.Transaction, 0, len(m.txMap)),
    }
    
    // Count by priority and collect transactions
    for _, tx := range m.txMap {
        stats.ByPriority[tx.Priority]++
        stats.PendingTxs = append(stats.PendingTxs, tx)
    }
    
    return stats
}

// Size returns current number of transactions in mempool
func (m *Mempool) Size() int {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    return len(m.txMap)
}

// GetTransaction retrieves a specific transaction by ID
func (m *Mempool) GetTransaction(txID string) (*types.Transaction, bool) {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    tx, exists := m.txMap[txID]
    return tx, exists
}

// RemoveTransaction removes a transaction from mempool (for failed transactions)
func (m *Mempool) RemoveTransaction(txID string) bool {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    if _, exists := m.txMap[txID]; exists {
        delete(m.txMap, txID)
        return true
    }
    return false
}

// Clear empties the mempool
func (m *Mempool) Clear() {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    m.transactions = &PriorityQueue{}
    heap.Init(m.transactions)
    m.txMap = make(map[string]*types.Transaction)
}

// GetPendingTransactionsByPriority returns transactions grouped by priority
func (m *Mempool) GetPendingTransactionsByPriority() map[int][]*types.Transaction {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    priorityMap := make(map[int][]*types.Transaction)
    
    for _, tx := range m.txMap {
        priorityMap[tx.Priority] = append(priorityMap[tx.Priority], tx)
    }
    
    return priorityMap
}
