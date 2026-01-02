package types

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