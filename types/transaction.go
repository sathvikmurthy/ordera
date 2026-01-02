package types

import "time"

// Transaction represents a unified transaction structure
// Replaces both TransactionRecord and PendingTransaction
type Transaction struct {
    ID        string    `json:"id"`         // Transaction ID
    TxID      string    `json:"txid"`       // Alias for ID (backward compatibility)
    From      string    `json:"from"`       // Sender address
    To        string    `json:"to"`         // Receiver address
    Amount    string    `json:"amount"`     // Transaction amount as string
    TxType    string    `json:"txType"`     // Transaction type: swap, borrow, lend, transfer
    Type      string    `json:"type"`       // Alias for TxType (backward compatibility)
    Priority  int       `json:"priority"`   // Priority: 0=highest (swap), 3=lowest (transfer)
    Timestamp time.Time `json:"timestamp"`  // Transaction timestamp
    Time      string    `json:"time"`       // Timestamp as string (backward compatibility)
    Status    string    `json:"status"`     // pending, processing, completed, failed
    GasFee    string    `json:"gasFee"`     // Gas fee for the transaction
}

// NewTransaction creates a new transaction with proper field mapping
func NewTransaction(id, from, to, amount, txType string, priority int) *Transaction {
    now := time.Now()
    return &Transaction{
        ID:        id,
        TxID:      id,
        From:      from,
        To:        to,
        Amount:    amount,
        TxType:    txType,
        Type:      txType,
        Priority:  priority,
        Timestamp: now,
        Time:      now.UTC().Format(time.RFC3339),
        Status:    "pending",
        GasFee:    "0.001", // Default gas fee
    }
}

// NewTransactionRecord creates a transaction record for chaincode (backward compatibility)
func NewTransactionRecord(txID, from, to, amount, txType string, priority int, timestamp string) Transaction {
    now := time.Now()
    if timestamp == "" {
        timestamp = now.UTC().Format(time.RFC3339)
    }
    
    return Transaction{
        ID:        txID,
        TxID:      txID,
        From:      from,
        To:        to,
        Amount:    amount,
        TxType:    txType,
        Type:      txType,
        Priority:  priority,
        Timestamp: now,
        Time:      timestamp,
        Status:    "completed",
        GasFee:    "0.001",
    }
}