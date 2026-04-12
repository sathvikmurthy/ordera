package types

// IncomingTransaction represents a transaction request from clients
type IncomingTransaction struct {
    From   string `json:"from"`
    To     string `json:"to"`
    Amount string `json:"amount"`
    TxType string `json:"txType"` // swap, borrow, lend, transfer
}

// TransactionResponse sent back to clients
type TransactionResponse struct {
    TransactionID string `json:"transactionId"`
    Status        string `json:"status"`
    Priority      int    `json:"priority"`
    GasFee        string `json:"gasFee"`
    Message       string `json:"message"`
}