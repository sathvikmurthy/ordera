package internal

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
}

// NewTransactionGateway creates a new gateway instance
func NewTransactionGateway(mempool *Mempool, batcher *Batcher) *TransactionGateway {
	return &TransactionGateway{
		mempool: mempool,
		batcher: batcher,
	}
}

// SubmitTransaction receives and queues a new transaction
func (tg *TransactionGateway) SubmitTransaction(incomingTx types.IncomingTransaction) (*types.TransactionResponse, error) {
	txID := tg.generateTransactionID(incomingTx)
	
	priority := types.GetPriorityByType(incomingTx.TxType)
	if priority == 99 {
		return nil, fmt.Errorf("invalid transaction type: %s", incomingTx.TxType)
	}
	
	tx := types.NewTransaction(txID, incomingTx.From, incomingTx.To, incomingTx.Amount, incomingTx.TxType, priority)

	gasFee := CalculateGasFee(incomingTx.TxType, tg.mempool.Utilization())
	tx.GasFee = fmt.Sprintf("%.6f", gasFee)
	GasFeeGauge.WithLabelValues(incomingTx.TxType).Set(gasFee)

	err := tg.mempool.AddTransaction(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to add transaction to mempool: %v", err)
	}

	log.Printf("📥 Transaction queued: %s (%s, priority: %d, gasFee: %s)",
		safeSubstring(txID, 8), incomingTx.TxType, priority, tx.GasFee)

	return &types.TransactionResponse{
		TransactionID: txID,
		Status:        "queued",
		Priority:      priority,
		GasFee:        tx.GasFee,
		Message:       fmt.Sprintf("Transaction queued with priority %d", priority),
	}, nil
}

// generateTransactionID creates a unique transaction ID
func (tg *TransactionGateway) generateTransactionID(tx types.IncomingTransaction) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%d", 
		tx.From, tx.To, tx.Amount, tx.TxType, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}

func safeSubstring(str string, maxLen int) string {
	if len(str) <= maxLen { return str }
	return str[:maxLen]
}

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

// HandleMempoolStatus shows the current size of the bucket system
func (tg *TransactionGateway) HandleMempoolStatus(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"total_mempool_size": tg.mempool.Size(),
		"priority_mapping": map[string]string{
			"0": "swap", "1": "borrow", "2": "lend", "3": "transfer",
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (tg *TransactionGateway) validateTransaction(tx types.IncomingTransaction) error {
	if tx.From == "" || tx.To == "" || tx.Amount == "" || tx.TxType == "" {
		return fmt.Errorf("missing required transaction fields")
	}
	return nil
}

// CORS middleware
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

// StartServer starts the HTTP server and registers endpoints
func (tg *TransactionGateway) StartServer(port string) {
	// Standard Transaction Routes
	http.HandleFunc("/submit", enableCORS(tg.HandleSubmitTransaction))
	http.HandleFunc("/mempool/status", enableCORS(tg.HandleMempoolStatus))
	
	// Health Check (Useful for Docker/Prometheus monitoring)
	http.HandleFunc("/health", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status":       "healthy",
			"mempool_size": tg.mempool.Size(),
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	
	log.Printf("🚀 Priority Gateway server active on port %s", port)
	log.Printf("📊 Metrics: http://localhost:%s/metrics", port)
	log.Printf("📥 Submit:  http://localhost:%s/submit", port)
	
	// nil means use the DefaultServeMux where /metrics was registered in server.go
	log.Fatal(http.ListenAndServe(":"+port, nil))
}