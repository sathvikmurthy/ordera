package types

// GetPriorityByType returns priority for transaction type
func GetPriorityByType(txType string) int {
    switch txType {
    case "swap":
        return 0 // Highest priority
    case "borrow":
        return 1
    case "lend":
        return 2
    case "transfer":
        return 3 // Lowest priority
    default:
        return 99 // Unknown types get lowest priority
    }
}

// ValidTransactionTypes returns a map of valid transaction types
func ValidTransactionTypes() map[string]bool {
    return map[string]bool{
        "swap":     true,
        "borrow":   true,
        "lend":     true,
        "transfer": true,
    }
}

// SetCompleted marks the transaction as completed
func (t *Transaction) SetCompleted() {
    t.Status = "completed"
}

// SetFailed marks the transaction as failed
func (t *Transaction) SetFailed() {
    t.Status = "failed"
}

// SetProcessing marks the transaction as being processed
func (t *Transaction) SetProcessing() {
    t.Status = "processing"
}