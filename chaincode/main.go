package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// TREASURY_ADDRESS is the fixed address that collects all gas fees on-chain
const TREASURY_ADDRESS = "network_treasury"

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
func parseDecimal(str string) (*big.Float, error) {
	f, ok := new(big.Float).SetString(str)
	if !ok {
		return nil, fmt.Errorf("invalid decimal format: %s", str)
	}
	return f, nil
}

// converts float back to string after calculations according to the struct's type
func floatToString(f *big.Float) string {
	return f.Text('f', -1)
}

// parseGasFee parses the gas fee string, falling back to 0.001 if invalid or empty
func parseGasFee(gasFeeStr string) *big.Float {
	if gasFeeStr == "" {
		return big.NewFloat(0.001)
	}
	f, err := parseDecimal(gasFeeStr)
	if err != nil {
		return big.NewFloat(0.001)
	}
	return f
}

// checks if wallet address exists
func (s *SmartContract) WalletExists(ctx contractapi.TransactionContextInterface, address string) (bool, error) {
	data, err := ctx.GetStub().GetState(address)
	return data != nil && err == nil, err
}

// ensureTreasury returns the treasury wallet, creating it if it doesn't exist
func (s *SmartContract) ensureTreasury(ctx contractapi.TransactionContextInterface) (*Wallet, error) {
	exists, err := s.WalletExists(ctx, TREASURY_ADDRESS)
	if err != nil {
		return nil, err
	}
	if !exists {
		treasury := Wallet{
			Address:       TREASURY_ADDRESS,
			Balance:       "0.0",
			History:       []Transaction{},
			TotalLent:     "0.0",
			TotalBorrowed: "0.0",
			OtherTokens:   make(map[string]Token),
		}
		if err := s.PutWallet(ctx, treasury); err != nil {
			return nil, fmt.Errorf("failed to create treasury wallet: %v", err)
		}
		return &treasury, nil
	}
	return s.GetWallet(ctx, TREASURY_ADDRESS)
}

// creditTreasury adds the fee amount to the treasury wallet balance
func (s *SmartContract) creditTreasury(ctx contractapi.TransactionContextInterface, fee *big.Float) error {
	treasury, err := s.ensureTreasury(ctx)
	if err != nil {
		return err
	}
	bal, _ := parseDecimal(treasury.Balance)
	bal.Add(bal, fee)
	treasury.Balance = floatToString(bal)
	return s.PutWallet(ctx, *treasury)
}

// creates a new wallet with initial values
func (s *SmartContract) CreateWallet(ctx contractapi.TransactionContextInterface) (string, error) {
	txID := ctx.GetStub().GetTxID()
	hash := sha256.Sum256([]byte(txID))
	address := hex.EncodeToString(hash[:])[:40]

	exists, err := s.WalletExists(ctx, address)
	if err != nil {
		return "", err
	}
	if exists {
		return "", fmt.Errorf("wallet already exists for this transaction")
	}

	wallet := Wallet{
		Address:       address,
		Balance:       "0.0",
		History:       []Transaction{},
		TotalLent:     "0.0",
		TotalBorrowed: "0.0",
		OtherTokens:   make(map[string]Token),
	}

	data, _ := json.Marshal(wallet)
	return address, ctx.GetStub().PutState(address, data)
}

// GetWallet retrieves wallet data from the ledger
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
		if err := json.Unmarshal(queryResponse.Value, &wallet); err != nil {
			continue
		}
		wallets = append(wallets, &wallet)
	}
	return wallets, nil
}

// GetTreasury returns the current treasury wallet (fee collector)
func (s *SmartContract) GetTreasury(ctx contractapi.TransactionContextInterface) (*Wallet, error) {
	return s.ensureTreasury(ctx)
}

// Transact routes a transaction to the correct handler based on txType.
// gasFee is the dynamically calculated fee from the gateway (sigmoid model).
func (s *SmartContract) Transact(ctx contractapi.TransactionContextInterface, from, to, amount, txType, gasFee string) error {
	fromWallet, err := s.GetWallet(ctx, from)
	if err != nil {
		return err
	}

	amountFloat, err := parseDecimal(amount)
	if err != nil {
		return fmt.Errorf("invalid amount: %v", err)
	}

	fabricTxID := ctx.GetStub().GetTxID()
	txTimestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("failed to get transaction timestamp: %v", err)
	}

	txID := fmt.Sprintf("tx_%s", fabricTxID[:16])
	timestamp := time.Unix(txTimestamp.Seconds, int64(txTimestamp.Nanos)).UTC().Format(time.RFC3339)
	priority := GetPriorityByType(txType)

	txRecord := NewTransactionRecord(txID, from, to, amount, txType, gasFee, priority, timestamp)

	switch priority {
	case 0:
		return s.handleSwap(ctx, fromWallet, to, amountFloat, gasFee, txRecord)
	case 1:
		return s.handleBorrow(ctx, fromWallet, to, amountFloat, gasFee, txRecord)
	case 2:
		return s.handleLend(ctx, fromWallet, to, amountFloat, gasFee, txRecord)
	case 3:
		return s.handleTransfer(ctx, fromWallet, to, amountFloat, gasFee, txRecord)
	default:
		return fmt.Errorf("invalid transaction type: %s", txType)
	}
}

// handleTransfer processes a transfer: deducts (amount + gasFee) from sender,
// credits amount to receiver, credits gasFee to treasury.
func (s *SmartContract) handleTransfer(ctx contractapi.TransactionContextInterface, fromWallet *Wallet, to string, amount *big.Float, gasFeeStr string, txRecord Transaction) error {
	toWallet, err := s.GetWallet(ctx, to)
	if err != nil {
		return err
	}

	fee := parseGasFee(gasFeeStr)
	fromBal, _ := parseDecimal(fromWallet.Balance)
	totalCost := new(big.Float).Add(amount, fee)

	if fromBal.Cmp(totalCost) < 0 {
		return fmt.Errorf("insufficient balance: need %s (amount + gas fee %s), have %s",
			floatToString(totalCost), floatToString(fee), fromWallet.Balance)
	}

	fromBal.Sub(fromBal, totalCost)
	toBal, _ := parseDecimal(toWallet.Balance)
	toBal.Add(toBal, amount)

	fromWallet.Balance = floatToString(fromBal)
	toWallet.Balance = floatToString(toBal)

	fromWallet.History = append(fromWallet.History, txRecord)
	toWallet.History = append(toWallet.History, txRecord)

	if err := s.creditTreasury(ctx, fee); err != nil {
		return fmt.Errorf("failed to credit treasury: %v", err)
	}
	s.PutWallet(ctx, *fromWallet)
	s.PutWallet(ctx, *toWallet)
	return nil
}

// handleLend processes a lend: sender lends amount to receiver, pays gas fee to treasury.
func (s *SmartContract) handleLend(ctx contractapi.TransactionContextInterface, fromWallet *Wallet, to string, amount *big.Float, gasFeeStr string, txRecord Transaction) error {
	toWallet, err := s.GetWallet(ctx, to)
	if err != nil {
		return err
	}

	fee := parseGasFee(gasFeeStr)
	fromBal, _ := parseDecimal(fromWallet.Balance)
	totalCost := new(big.Float).Add(amount, fee)

	if fromBal.Cmp(totalCost) < 0 {
		return fmt.Errorf("insufficient balance: need %s (amount + gas fee %s), have %s",
			floatToString(totalCost), floatToString(fee), fromWallet.Balance)
	}

	fromBal.Sub(fromBal, totalCost)
	toBal, _ := parseDecimal(toWallet.Balance)
	toBal.Add(toBal, amount)

	fromLent, _ := parseDecimal(fromWallet.TotalLent)
	fromLent.Add(fromLent, amount)
	toBorrowed, _ := parseDecimal(toWallet.TotalBorrowed)
	toBorrowed.Add(toBorrowed, amount)

	fromWallet.Balance = floatToString(fromBal)
	fromWallet.TotalLent = floatToString(fromLent)
	toWallet.Balance = floatToString(toBal)
	toWallet.TotalBorrowed = floatToString(toBorrowed)

	fromWallet.History = append(fromWallet.History, txRecord)
	toWallet.History = append(toWallet.History, txRecord)

	if err := s.creditTreasury(ctx, fee); err != nil {
		return fmt.Errorf("failed to credit treasury: %v", err)
	}
	s.PutWallet(ctx, *fromWallet)
	s.PutWallet(ctx, *toWallet)
	return nil
}

// handleBorrow processes a borrow: borrower receives amount, pays gas fee to treasury.
func (s *SmartContract) handleBorrow(ctx contractapi.TransactionContextInterface, fromWallet *Wallet, to string, amount *big.Float, gasFeeStr string, txRecord Transaction) error {
	toWallet, err := s.GetWallet(ctx, to)
	if err != nil {
		return err
	}

	fee := parseGasFee(gasFeeStr)
	fromBal, _ := parseDecimal(fromWallet.Balance)

	if fromBal.Cmp(fee) < 0 {
		return fmt.Errorf("insufficient balance for gas fee: need %s, have %s",
			floatToString(fee), fromWallet.Balance)
	}

	// Borrower only pays the gas fee; they receive the amount from the lender (to)
	lenderBal, _ := parseDecimal(toWallet.Balance)
	if lenderBal.Cmp(amount) < 0 {
		return fmt.Errorf("lender has insufficient balance: need %s, have %s",
			floatToString(amount), toWallet.Balance)
	}

	fromBal.Sub(fromBal, fee)
	lenderBal.Sub(lenderBal, amount)

	borrowerReceived := new(big.Float).Set(amount)
	fromNewBal := new(big.Float).Add(fromBal, borrowerReceived)

	fromWallet.Balance = floatToString(fromNewBal)
	toWallet.Balance = floatToString(lenderBal)

	fromBorrowed, _ := parseDecimal(fromWallet.TotalBorrowed)
	fromBorrowed.Add(fromBorrowed, amount)
	toLent, _ := parseDecimal(toWallet.TotalLent)
	toLent.Add(toLent, amount)

	fromWallet.TotalBorrowed = floatToString(fromBorrowed)
	toWallet.TotalLent = floatToString(toLent)

	fromWallet.History = append(fromWallet.History, txRecord)
	toWallet.History = append(toWallet.History, txRecord)

	if err := s.creditTreasury(ctx, fee); err != nil {
		return fmt.Errorf("failed to credit treasury: %v", err)
	}
	s.PutWallet(ctx, *fromWallet)
	s.PutWallet(ctx, *toWallet)
	return nil
}

// handleSwap processes a swap: deducts (amount + gasFee) from sender,
// credits amount to receiver, credits gasFee to treasury.
func (s *SmartContract) handleSwap(ctx contractapi.TransactionContextInterface, fromWallet *Wallet, to string, amount *big.Float, gasFeeStr string, txRecord Transaction) error {
	return s.handleTransfer(ctx, fromWallet, to, amount, gasFeeStr, txRecord)
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
