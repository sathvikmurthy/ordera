package main

import (
    "fmt"
    "log"
    "sync"
    "time"
    
    "priority-fabric-project/types"
)

// Batcher manages batching transactions from mempool
type Batcher struct {
    mempool         *Mempool
    batchSize       int
    batchTimeout    time.Duration
    processor       BatchProcessor
    stopChan        chan bool
    running         bool
    batchCount      int
    totalProcessed  int
    completedTxs    []*types.Transaction // Store completed transactions
    wsHub           *WebSocketHub
    mutex           sync.RWMutex
}

// BatchProcessor interface for processing transaction batches
type BatchProcessor interface {
    ProcessBatch(batch []*types.Transaction) error
}

// FabricBatchProcessor processes batches by sending to Hyperledger Fabric
type FabricBatchProcessor struct {
    fabricClient  *FabricClient
    chaincodeName string
    channelName   string
}

// NewBatcher creates a new batcher instance
func NewBatcher(mempool *Mempool, batchSize int, batchTimeout time.Duration, fabricClient *FabricClient, wsHub *WebSocketHub) *Batcher {
    return &Batcher{
        mempool:      mempool,
        batchSize:    batchSize,
        batchTimeout: batchTimeout,
        processor:    &FabricBatchProcessor{
            fabricClient:  fabricClient,
            chaincodeName: "wallet",
            channelName:   "mychannel",
        },
        stopChan:     make(chan bool),
        running:      false,
        batchCount:   0,
        totalProcessed: 0,
        wsHub:        wsHub,
    }
}

// Start begins the batching process
func (b *Batcher) Start() {
    if b.running {
        log.Println("Batcher is already running")
        return
    }
    
    b.running = true
    log.Printf("🚀 Starting batcher: batchSize=%d, timeout=%v", b.batchSize, b.batchTimeout)
    
    ticker := time.NewTicker(b.batchTimeout)
    defer ticker.Stop()
    
    // Countdown ticker (1 second intervals)
    countdownTicker := time.NewTicker(1 * time.Second)
    defer countdownTicker.Stop()
    
    lastBatchTime := time.Now()
    
    for {
        select {
        case <-b.stopChan:
            log.Println("🛑 Batcher stopped")
            b.running = false
            return
            
        case <-ticker.C:
            // Time-based batching
            if b.mempool.Size() > 0 {
                b.processBatch("timeout")
                lastBatchTime = time.Now()
            }
            
        case <-countdownTicker.C:
            // Show countdown if there are pending transactions
            if b.mempool.Size() > 0 {
                elapsed := time.Since(lastBatchTime)
                remaining := b.batchTimeout - elapsed
                if remaining > 0 {
                    log.Printf("⏱️  Waiting for batch... %d transactions queued, %.1fs remaining (or %d more txs needed)",
                        b.mempool.Size(), remaining.Seconds(), b.batchSize-b.mempool.Size())
                    
                    // Broadcast countdown event
                    if b.wsHub != nil {
                        stats := b.mempool.GetStats()
                        b.wsHub.BroadcastEvent(EventBatchCountdown, map[string]interface{}{
                            "remainingSeconds": int(remaining.Seconds()),
                            "mempoolSize":      b.mempool.Size(),
                            "batchSize":        b.batchSize,
                            "mempoolStats":     stats,
                        })
                    }
                }
            }
            
        default:
            // Size-based batching
            if b.mempool.Size() >= b.batchSize {
                b.processBatch("size-limit")
                lastBatchTime = time.Now()
            }
            
            // Small sleep to prevent busy waiting
            time.Sleep(100 * time.Millisecond)
        }
    }
}

// Stop stops the batcher
func (b *Batcher) Stop() {
    if b.running {
        b.stopChan <- true
    }
}

// processBatch extracts and processes a batch of transactions
func (b *Batcher) processBatch(trigger string) {
    b.batchCount++
    
    // Alternate between standard and quota-based modes
    // Odd batches (1, 3, 5...): Standard priority ordering
    // Even batches (2, 4, 6...): Quota-based anti-starvation
    useQuotaMode := (b.batchCount % 2) == 0
    
    batch := b.mempool.GetBatch(b.batchSize, useQuotaMode)
    
    if len(batch) == 0 {
        b.batchCount-- // Decrement if no batch was created
        return
    }
    
    // Calculate priority distribution
    priorityCount := make(map[int]int)
    for _, tx := range batch {
        priorityCount[tx.Priority]++
    }
    
    // Determine batch mode label
    batchMode := "STANDARD (Priority-Only)"
    if useQuotaMode {
        batchMode = "QUOTA-BASED (Anti-Starvation)"
    }
    
    log.Printf("⚡ Processing batch #%d of %d transactions (trigger: %s, mode: %s)", 
        b.batchCount, len(batch), trigger, batchMode)
    log.Printf("🎯 Priority distribution: swap:%d, borrow:%d, lend:%d, transfer:%d", 
        priorityCount[0], priorityCount[1], priorityCount[2], priorityCount[3])
    
    // Broadcast batch started event
    if b.wsHub != nil {
        b.wsHub.BroadcastEvent(EventBatchStarted, map[string]interface{}{
            "batchNumber":    b.batchCount,
            "batchSize":      len(batch),
            "trigger":        trigger,
            "mode":           batchMode,
            "useQuotaMode":   useQuotaMode,
            "priorityCount":  priorityCount,
        })
    }
    
    // Log batch details for debugging
    log.Printf("📦 Batch contents:")
    for i, tx := range batch {
        log.Printf("  [%d] %s: %s -> %s (%s %s) [Priority: %d - %s]", 
            i+1, safeSubstring(tx.ID, 8), safeSubstring(tx.From, 8), safeSubstring(tx.To, 8), 
            tx.Amount, tx.TxType, tx.Priority, getPriorityName(tx.Priority))
    }
    
    // Process the batch
    err := b.processor.ProcessBatch(batch)
    if err != nil {
        log.Printf("❌ Failed to process batch #%d: %v", b.batchCount, err)
        b.handleFailedBatch(batch, err)
    } else {
        b.totalProcessed += len(batch)
        
        // Store completed transactions
        b.mutex.Lock()
        b.completedTxs = append(b.completedTxs, batch...)
        b.mutex.Unlock()
        
        log.Printf("✅ Successfully processed batch #%d (%d transactions). Total processed: %d", 
            b.batchCount, len(batch), b.totalProcessed)
        
        // Broadcast batch completed event
        if b.wsHub != nil {
            b.wsHub.BroadcastEvent(EventBatchCompleted, map[string]interface{}{
                "batchNumber":      b.batchCount,
                "batchSize":        len(batch),
                "totalProcessed":   b.totalProcessed,
                "priorityCount":    priorityCount,
                "mode":             batchMode,
            })
        }
    }
}

// GetCompletedTransactions returns all completed transactions
func (b *Batcher) GetCompletedTransactions() []*types.Transaction {
    b.mutex.RLock()
    defer b.mutex.RUnlock()
    
    // Return a copy to prevent external modification
    completed := make([]*types.Transaction, len(b.completedTxs))
    copy(completed, b.completedTxs)
    return completed
}

// getPriorityName returns the name for a priority level
func getPriorityName(priority int) string {
    switch priority {
    case 0:
        return "swap"
    case 1:
        return "borrow"
    case 2:
        return "lend"
    case 3:
        return "transfer"
    default:
        return "unknown"
    }
}

// handleFailedBatch handles failed batch processing
func (b *Batcher) handleFailedBatch(batch []*types.Transaction, err error) {
    log.Printf("🔧 Handling failed batch of %d transactions", len(batch))
    
    // Mark transactions as failed
    for _, tx := range batch {
        tx.SetFailed()
        log.Printf("❌ Transaction %s marked as failed: %v", safeSubstring(tx.ID, 8), err)
    }
    
    // In production, you might want to:
    // 1. Re-queue transactions with exponential backoff
    // 2. Send to dead letter queue
    // 3. Notify monitoring systems
}

// ProcessBatch implementation for Fabric
// This submits each transaction through Fabric's consensus mechanism
func (fbp *FabricBatchProcessor) ProcessBatch(batch []*types.Transaction) error {
    log.Printf("🔗 Submitting batch of %d transactions to Fabric chaincode '%s'", 
        len(batch), fbp.chaincodeName)
    
    // Check if Fabric client is available
    if fbp.fabricClient == nil {
        log.Printf("⚠️  Fabric client not initialized, running in simulation mode")
        return fbp.simulateProcessBatch(batch)
    }
    
    for _, tx := range batch {
        log.Printf("📤 Submitting to Fabric: %s (%s, priority: %d)", 
            safeSubstring(tx.ID, 8), tx.TxType, tx.Priority)
        
        // Mark as processing
        tx.SetProcessing()
        
        // Submit transaction to Fabric network
        // This goes through the full consensus process:
        // 1. Proposal sent to endorsing peers
        // 2. Endorsements collected
        // 3. Transaction submitted to orderer
        // 4. Orderer creates block and broadcasts
        // 5. Peers validate and commit to ledger
        result, err := fbp.fabricClient.SubmitTransaction(
            "Transact",      // Chaincode function name
            tx.From,         // From address
            tx.To,           // To address
            tx.Amount,       // Amount
            tx.TxType,       // Transaction type
        )
        
        if err != nil {
            log.Printf("❌ Fabric transaction %s failed: %v", safeSubstring(tx.ID, 8), err)
            tx.SetFailed()
            return fmt.Errorf("failed to submit transaction %s to Fabric: %w", tx.ID, err)
        }
        
        tx.SetCompleted()
        log.Printf("✅ Transaction %s committed to Fabric ledger (result: %s)", 
            safeSubstring(tx.ID, 8), string(result))
        
        // Broadcast transaction status change (in FabricBatchProcessor context, access via batch processor)
        // Note: We'll handle this in the Batcher's processBatch method instead
    }
    
    log.Printf("🎉 Batch of %d transactions successfully committed to Fabric ledger", len(batch))
    return nil
}

// simulateProcessBatch simulates batch processing when Fabric client is not available
func (fbp *FabricBatchProcessor) simulateProcessBatch(batch []*types.Transaction) error {
    log.Printf("🧪 Simulating batch processing (Fabric client not connected)")
    
    for _, tx := range batch {
        log.Printf("📤 Simulating: %s (%s, priority: %d)", 
            safeSubstring(tx.ID, 8), tx.TxType, tx.Priority)
        
        // Simulate processing time based on priority
        processingTime := time.Duration(tx.Priority+1) * 10 * time.Millisecond
        time.Sleep(processingTime)
        
        tx.SetCompleted()
        log.Printf("✅ Simulated transaction %s completed", safeSubstring(tx.ID, 8))
    }
    
    return nil
}

// GetBatcherStats returns current batcher statistics
func (b *Batcher) GetBatcherStats() types.BatcherStats {
    return types.BatcherStats{
        Running:      b.running,
        BatchSize:    b.batchSize,
        BatchTimeout: b.batchTimeout.String(),
        MempoolSize:  b.mempool.Size(),
    }
}

// GetDetailedStats returns detailed batcher statistics
func (b *Batcher) GetDetailedStats() map[string]interface{} {
    return map[string]interface{}{
        "running":         b.running,
        "batchSize":       b.batchSize,
        "batchTimeout":    b.batchTimeout.String(),
        "mempoolSize":     b.mempool.Size(),
        "batchesProcessed": b.batchCount,
        "totalTransactionsProcessed": b.totalProcessed,
        "averageTransactionsPerBatch": func() float64 {
            if b.batchCount == 0 {
                return 0
            }
            return float64(b.totalProcessed) / float64(b.batchCount)
        }(),
    }
}
