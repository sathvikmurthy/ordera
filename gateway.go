package main

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
    
    "priority-fabric-project/types"
)

// TransactionGateway receives incoming transactions and adds them to mempool
type TransactionGateway struct {
    mempool *Mempool
    batcher *Batcher
    wsHub   *WebSocketHub
}

// NewTransactionGateway creates a new gateway instance
func NewTransactionGateway(mempool *Mempool, batcher *Batcher, wsHub *WebSocketHub) *TransactionGateway {
    return &TransactionGateway{
        mempool: mempool,
        batcher: batcher,
        wsHub:   wsHub,
    }
}

// SubmitTransaction receives and queues a new transaction
func (tg *TransactionGateway) SubmitTransaction(incomingTx types.IncomingTransaction) (*types.TransactionResponse, error) {
    // Generate transaction ID
    txID := tg.generateTransactionID(incomingTx)
    
    // Get priority for transaction type
    priority := types.GetPriorityByType(incomingTx.TxType)
    if priority == 99 {
        return nil, fmt.Errorf("invalid transaction type: %s", incomingTx.TxType)
    }
    
    // Create transaction using shared type
    tx := types.NewTransaction(txID, incomingTx.From, incomingTx.To, incomingTx.Amount, incomingTx.TxType, priority)
    
    // Add to mempool
    err := tg.mempool.AddTransaction(tx)
    if err != nil {
        return nil, fmt.Errorf("failed to add transaction to mempool: %v", err)
    }
    
    log.Printf("📥 Transaction queued: %s (%s, priority: %d, from: %s, to: %s, amount: %s)", 
        safeSubstring(txID, 8), incomingTx.TxType, priority, 
        safeSubstring(incomingTx.From, 8), safeSubstring(incomingTx.To, 8), incomingTx.Amount)
    
    // Broadcast transaction submitted event
    if tg.wsHub != nil {
        tg.wsHub.BroadcastEvent(EventTxSubmitted, map[string]interface{}{
            "transactionId": txID,
            "from":          incomingTx.From,
            "to":            incomingTx.To,
            "amount":        incomingTx.Amount,
            "txType":        incomingTx.TxType,
            "priority":      priority,
            "status":        "queued",
        })
    }
    
    return &types.TransactionResponse{
        TransactionID: txID,
        Status:        "queued",
        Priority:      priority,
        Message:       fmt.Sprintf("Transaction queued with priority %d (%s transactions processed first)", priority, incomingTx.TxType),
    }, nil
}

// generateTransactionID creates a unique transaction ID
func (tg *TransactionGateway) generateTransactionID(tx types.IncomingTransaction) string {
    data := fmt.Sprintf("%s:%s:%s:%s:%d", 
        tx.From, tx.To, tx.Amount, tx.TxType, time.Now().UnixNano())
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])[:16] // Use first 16 chars for readability
}

// safeSubstring safely truncates a string to maxLen without panicking
func safeSubstring(str string, maxLen int) string {
    if len(str) <= maxLen {
        return str
    }
    return str[:maxLen]
}

// HTTP Handlers

// HandleSubmitTransaction HTTP handler for submitting transactions
func (tg *TransactionGateway) HandleSubmitTransaction(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var incomingTx types.IncomingTransaction
    if err := json.NewDecoder(r.Body).Decode(&incomingTx); err != nil {
        http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
        return
    }
    
    // Validate transaction
    if err := tg.validateTransaction(incomingTx); err != nil {
        http.Error(w, "Invalid transaction: "+err.Error(), http.StatusBadRequest)
        return
    }
    
    response, err := tg.SubmitTransaction(incomingTx)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// HandleMempoolStatus HTTP handler for mempool status
func (tg *TransactionGateway) HandleMempoolStatus(w http.ResponseWriter, r *http.Request) {
    stats := tg.mempool.GetStats()
    
    // Add some additional useful information
    response := map[string]interface{}{
        "stats": stats,
        "priorityBreakdown": map[string]string{
            "0": "swap (highest priority)",
            "1": "borrow",
            "2": "lend", 
            "3": "transfer (lowest priority)",
        },
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// HandleBatcherStatus HTTP handler for batcher status
func (tg *TransactionGateway) HandleBatcherStatus(w http.ResponseWriter, r *http.Request) {
    stats := tg.batcher.GetDetailedStats()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

// HandleTransactionStatus HTTP handler for checking specific transaction status
func (tg *TransactionGateway) HandleTransactionStatus(w http.ResponseWriter, r *http.Request) {
    txID := r.URL.Query().Get("id")
    if txID == "" {
        http.Error(w, "Transaction ID is required", http.StatusBadRequest)
        return
    }
    
    tx, exists := tg.mempool.GetTransaction(txID)
    if !exists {
        http.Error(w, "Transaction not found", http.StatusNotFound)
        return
    }
    
    response := map[string]interface{}{
        "transaction": tx,
        "queuePosition": tg.calculateQueuePosition(tx),
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// calculateQueuePosition estimates the position of a transaction in the queue
func (tg *TransactionGateway) calculateQueuePosition(tx *types.Transaction) int {
    position := 1
    priorityTxs := tg.mempool.GetPendingTransactionsByPriority()
    
    // Count transactions with higher priority
    for priority := 0; priority < tx.Priority; priority++ {
        position += len(priorityTxs[priority])
    }
    
    // Count transactions with same priority but earlier timestamp
    for _, samePriorityTx := range priorityTxs[tx.Priority] {
        if samePriorityTx.Timestamp.Before(tx.Timestamp) {
            position++
        }
    }
    
    return position
}

// validateTransaction validates incoming transaction data
func (tg *TransactionGateway) validateTransaction(tx types.IncomingTransaction) error {
    if tx.From == "" {
        return fmt.Errorf("from address is required")
    }
    if tx.To == "" {
        return fmt.Errorf("to address is required")
    }
    if tx.Amount == "" {
        return fmt.Errorf("amount is required")
    }
    if tx.TxType == "" {
        return fmt.Errorf("transaction type is required")
    }
    
    // Validate transaction type using shared validation
    validTypes := types.ValidTransactionTypes()
    if !validTypes[tx.TxType] {
        return fmt.Errorf("invalid transaction type: %s. Valid types: swap, borrow, lend, transfer", tx.TxType)
    }
    
    return nil
}

// HandleCompletedTransactions HTTP handler for viewing completed transactions
func (tg *TransactionGateway) HandleCompletedTransactions(w http.ResponseWriter, r *http.Request) {
    completed := tg.batcher.GetCompletedTransactions()
    
    // Group by priority for analysis
    byPriority := make(map[int]int)
    for _, tx := range completed {
        byPriority[tx.Priority]++
    }
    
    response := map[string]interface{}{
        "totalCompleted": len(completed),
        "byPriority": map[string]int{
            "swap":     byPriority[0],
            "borrow":   byPriority[1],
            "lend":     byPriority[2],
            "transfer": byPriority[3],
        },
        "transactions": completed,
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// CORS middleware
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        // Handle preflight requests
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next(w, r)
    }
}

// StartServer starts the HTTP server
func (tg *TransactionGateway) StartServer(port string) {
    // WebSocket endpoint
    if tg.wsHub != nil {
        http.HandleFunc("/ws", tg.wsHub.HandleWebSocket)
    }
    
    http.HandleFunc("/submit", enableCORS(tg.HandleSubmitTransaction))
    http.HandleFunc("/mempool/status", enableCORS(tg.HandleMempoolStatus))
    http.HandleFunc("/batcher/status", enableCORS(tg.HandleBatcherStatus))
    http.HandleFunc("/transaction/status", enableCORS(tg.HandleTransactionStatus))
    http.HandleFunc("/transactions/completed", enableCORS(tg.HandleCompletedTransactions))
    
    // Add a simple health check endpoint
    http.HandleFunc("/health", enableCORS(func(w http.ResponseWriter, r *http.Request) {
        response := map[string]interface{}{
            "status": "healthy",
            "service": "Priority Transaction Gateway",
            "timestamp": time.Now().UTC().Format(time.RFC3339),
            "mempool_size": tg.mempool.Size(),
            "batcher_running": tg.batcher.running,
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }))
    
    log.Printf("🚀 Priority Transaction Gateway server starting on port %s", port)
    log.Printf("📡 Available endpoints:")
    if tg.wsHub != nil {
        log.Printf("   WS   /ws - WebSocket connection for real-time updates")
    }
    log.Printf("   POST /submit - Submit new transactions")
    log.Printf("   GET  /mempool/status - View mempool statistics")
    log.Printf("   GET  /batcher/status - View batcher status")
    log.Printf("   GET  /transaction/status?id=<txid> - Check transaction status")
    log.Printf("   GET  /transactions/completed - View all completed transactions")
    log.Printf("   GET  /health - Health check")
    log.Printf("")
    log.Printf("💡 Priority levels: swap(0) > borrow(1) > lend(2) > transfer(3)")
    
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
