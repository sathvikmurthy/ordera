package main

import "time"

// Wallet represents a user's wallet with balance and transaction history
type Wallet struct {
	Address       string              `json:"address"`
	Balance       string              `json:"balance"` // base token (gas fees gets charged)
	History       []Transaction       `json:"history"`
	TotalLent     string              `json:"totalLent"`
	TotalBorrowed string              `json:"totalBorrowed"`
	OtherTokens   map[string]Token    `json:"otherTokens"`
}

// Token represents other tokens held in the wallet
type Token struct {
	Ticker string `json:"ticker"`
	Name   string `json:"name"`
	Amount string `json:"amount"`
}

// Transaction represents a unified transaction structure
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

// NewTransactionRecord creates a transaction record for chaincode
func NewTransactionRecord(txID, from, to, amount, txType, gasFee string, priority int, timestamp string) Transaction {
	var parsedTime time.Time
	if timestamp != "" {
		parsedTime, _ = time.Parse(time.RFC3339, timestamp)
	} else {
		parsedTime = time.Unix(0, 0)
	}
	if gasFee == "" {
		gasFee = "0.001"
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
		Timestamp: parsedTime,
		Time:      timestamp,
		Status:    "completed",
		GasFee:    gasFee,
	}
}

// GetPriorityByType returns the priority number from transaction type
func GetPriorityByType(txType string) int {
	switch txType {
	case "swap":
		return 0 // highest priority
	case "borrow":
		return 1
	case "lend":
		return 2
	case "transfer":
		return 3 // lowest priority
	default:
		return 3
	}
}
