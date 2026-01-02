package main

import (
    "crypto/sha256"
    "encoding/json"
    "encoding/hex"
    "math/big"
    "fmt"
    "time"
    "log"

    "github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
    contractapi.Contract
}

// marshals wallet data into the ledger
func (s *SmartContract) PutWallet(ctx contractapi.TransactionContextInterface, wallet Wallet) error {
    walletJSON, err := json.Marshal(wallet)
    if err != nil {
        return fmt.Errorf("failed to marshal wallet: %v", err)
    }

    return ctx.GetStub().PutState(wallet.Address, walletJSON)
}

// converts the string in structs to float for calculations in functions
func parseDecimal(s string) (*big.Float, error) {
    f, ok := new(big.Float).SetString(s)
    if !ok {
        return nil, fmt.Errorf("invalid decimal format: %s", s)
    }
    return f, nil
}

// converts float back to string after calculations according to the struct's type
func floatToString(f *big.Float) string {
    return f.Text('f', -1)
}


// checks if wallet address exists
func (s *SmartContract) WalletExists(ctx contractapi.TransactionContextInterface, address string) (bool, error) {
    data, err := ctx.GetStub().GetState(address)
    return data != nil && err == nil, err
}

// creates a new wallet with initial values
// Uses transaction ID to ensure deterministic wallet address across all peers
func (s *SmartContract) CreateWallet(ctx contractapi.TransactionContextInterface) (string, error) {
    // Use transaction ID to create deterministic wallet address
    // This ensures all endorsing peers produce the same result
    txID := ctx.GetStub().GetTxID()
    hash := sha256.Sum256([]byte(txID))
    address := hex.EncodeToString(hash[:])[:40]

    // Check if wallet already exists
    exists, err := s.WalletExists(ctx, address)
    if err != nil {
        return "", err
    }
    if exists {
        return "", fmt.Errorf("wallet already exists for this transaction")
    }

    wallet := Wallet{
        Address: address,
        Balance: "0.0",
        History: []Transaction{},
        TotalLent: "0.0",
        TotalBorrowed: "0.0",
        OtherTokens: make(map[string]Token),
    }

    data, _ := json.Marshal(wallet)
    return address, ctx.GetStub().PutState(address, data)
}

// GetWallet retrieves wallet data from the global state by making use of the GetState(address) function
func (s *SmartContract) GetWallet(ctx contractapi.TransactionContextInterface, address string) (*Wallet, error) {
    data, err := ctx.GetStub().GetState(address)
    if err != nil || data == nil {
        return nil, fmt.Errorf("wallet not found: %s", address)
    }
    var wallet Wallet
    err = json.Unmarshal(data, &wallet)
    return &wallet, err
}

// AddBalance adds funds to a wallet (for testing/initial funding)
func (s *SmartContract) AddBalance(ctx contractapi.TransactionContextInterface, address string, amount string) error {
    wallet, err := s.GetWallet(ctx, address)
    if err != nil {
        return err
    }
    
    currentBal, err := parseDecimal(wallet.Balance)
    if err != nil {
        return err
    }
    
    addAmount, err := parseDecimal(amount)
    if err != nil {
        return err
    }
    
    if addAmount.Cmp(big.NewFloat(0)) <= 0 {
        return fmt.Errorf("amount must be positive")
    }
    
    currentBal.Add(currentBal, addAmount)
    wallet.Balance = floatToString(currentBal)
    
    return s.PutWallet(ctx, *wallet)
}

// GetAllWallets retrieves all wallets from the ledger
func (s *SmartContract) GetAllWallets(ctx contractapi.TransactionContextInterface) ([]*Wallet, error) {
    // Create an iterator to get all wallet data from the ledger
    // Empty string as start and end key means we get all keys
    resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
    if err != nil {
        return nil, fmt.Errorf("failed to get state by range: %v", err)
    }
    defer resultsIterator.Close()

    var wallets []*Wallet
    
    for resultsIterator.HasNext() {
        queryResponse, err := resultsIterator.Next()
        if err != nil {
            return nil, fmt.Errorf("failed to iterate: %v", err)
        }

        var wallet Wallet
        err = json.Unmarshal(queryResponse.Value, &wallet)
        if err != nil {
            // Skip entries that aren't wallets (might be other data)
            continue
        }

        wallets = append(wallets, &wallet)
    }

    return wallets, nil
}

// transact tokens or base currency with swap, borrow, lend, transfer types
func (s *SmartContract) Transact(ctx contractapi.TransactionContextInterface, from string, to string, amount string, txType string) error {
    fromWallet, err := s.GetWallet(ctx, from)
    if err != nil {
        return err
    }

    amountFloat, err := parseDecimal(amount)
    if err != nil {
        return fmt.Errorf("invalid amount")
    }

    // Use transaction ID and timestamp from Fabric (deterministic across peers)
    fabricTxID := ctx.GetStub().GetTxID()
    txTimestamp, err := ctx.GetStub().GetTxTimestamp()
    if err != nil {
        return fmt.Errorf("failed to get transaction timestamp: %v", err)
    }
    
    // Create deterministic transaction ID from Fabric tx ID
    txID := fmt.Sprintf("tx_%s", fabricTxID[:16])
    timestamp := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC().Format(time.RFC3339)
    priority := GetPriorityByType(txType)

    // Create transaction record
    txRecord := NewTransactionRecord(txID, from, to, amount, txType, priority, timestamp)

    switch priority {
    case 0: // swap
        return s.handleSwap(ctx, fromWallet, to, amountFloat, txRecord)
    case 1: // borrow
        return s.handleBorrow(ctx, fromWallet, to, amountFloat, txRecord)
    case 2: // lend
        return s.handleLend(ctx, fromWallet, to, amountFloat, txRecord)
    case 3: // transfer
        return s.handleTransfer(ctx, fromWallet, to, amountFloat, txRecord)
    default:
        return fmt.Errorf("invalid transaction type")
    }
}

// Handle transfer transactions (priority 3)
func (s *SmartContract) handleTransfer(ctx contractapi.TransactionContextInterface, fromWallet *Wallet, to string, amount *big.Float, txRecord Transaction) error {
    gasFee := big.NewFloat(0.001)
    
    toWallet, err := s.GetWallet(ctx, to)
    if err != nil {
        return err
    }
    
    fromBal, _ := parseDecimal(fromWallet.Balance)
    totalCost := new(big.Float).Add(amount, gasFee)

    if fromBal.Cmp(totalCost) < 0 {
        return fmt.Errorf("insufficient balance")
    }

    fromBal.Sub(fromBal, totalCost)
    toBal, _ := parseDecimal(toWallet.Balance)
    toBal.Add(toBal, amount)

    fromWallet.Balance = floatToString(fromBal)
    toWallet.Balance = floatToString(toBal)

    // Add to history
    fromWallet.History = append(fromWallet.History, txRecord)
    toWallet.History = append(toWallet.History, txRecord)

    s.PutWallet(ctx, *fromWallet)
    s.PutWallet(ctx, *toWallet)
    
    return nil
}

// Handle lend transactions (priority 2)
func (s *SmartContract) handleLend(ctx contractapi.TransactionContextInterface, fromWallet *Wallet, to string, amount *big.Float, txRecord Transaction) error {
    toWallet, err := s.GetWallet(ctx, to)
    if err != nil {
        return err
    }

    fromBal, _ := parseDecimal(fromWallet.Balance)
    if fromBal.Cmp(amount) < 0 {
        return fmt.Errorf("insufficient balance to lend")
    }

    // Update balances
    fromBal.Sub(fromBal, amount)
    toBal, _ := parseDecimal(toWallet.Balance)
    toBal.Add(toBal, amount)

    // Update lending records
    fromLent, _ := parseDecimal(fromWallet.TotalLent)
    fromLent.Add(fromLent, amount)
    
    toBorrowed, _ := parseDecimal(toWallet.TotalBorrowed)
    toBorrowed.Add(toBorrowed, amount)

    fromWallet.Balance = floatToString(fromBal)
    fromWallet.TotalLent = floatToString(fromLent)
    toWallet.Balance = floatToString(toBal)
    toWallet.TotalBorrowed = floatToString(toBorrowed)

    // Add to history
    fromWallet.History = append(fromWallet.History, txRecord)
    toWallet.History = append(toWallet.History, txRecord)

    s.PutWallet(ctx, *fromWallet)
    s.PutWallet(ctx, *toWallet)
    
    return nil
}

// Handle borrow transactions (priority 1)
func (s *SmartContract) handleBorrow(ctx contractapi.TransactionContextInterface, fromWallet *Wallet, to string, amount *big.Float, txRecord Transaction) error {
    // Similar to lend but reverse the roles
    return s.handleLend(ctx, fromWallet, to, amount, txRecord)
}

// Handle swap transactions (priority 0 - highest)
func (s *SmartContract) handleSwap(ctx contractapi.TransactionContextInterface, fromWallet *Wallet, to string, amount *big.Float, txRecord Transaction) error {
    // For now, treat as high priority transfer
    return s.handleTransfer(ctx, fromWallet, to, amount, txRecord)
}

func main() {
    assetChaincode, err := contractapi.NewChaincode(&SmartContract{})
    if err != nil {
        log.Panicf("Error creating asset-transfer-basic chaincode: %v", err)
    }

    if err := assetChaincode.Start(); err != nil {
        log.Panicf("Error starting asset-transfer-basic chaincode: %v", err)
    }
}
