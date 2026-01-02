package main

import (
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// FabricClient manages connection to Hyperledger Fabric network
type FabricClient struct {
	gateway       *client.Gateway
	network       *client.Network
	contract      *client.Contract
	grpcConn      *grpc.ClientConn
	channelName   string
	chaincodeName string
}

// FabricConfig holds configuration for connecting to Fabric network
type FabricConfig struct {
	// Path to the peer's TLS certificate
	TLSCertPath string
	
	// Peer endpoint (e.g., "localhost:7051")
	PeerEndpoint string
	
	// MSP ID for the organization (e.g., "Org1MSP")
	MSPID string
	
	// Path to the user's certificate
	CertPath string
	
	// Path to the user's private key
	KeyPath string
	
	// Channel name (e.g., "mychannel")
	ChannelName string
	
	// Chaincode name (e.g., "wallet")
	ChaincodeName string
}

// NewFabricClient creates and initializes a new Fabric client
// This establishes a gRPC connection to the Fabric peer and creates a gateway
func NewFabricClient(config FabricConfig) (*FabricClient, error) {
	// Step 1: Load the TLS certificate for secure communication with the peer
	// The peer's TLS cert ensures we're connecting to the right peer
	tlsCert, err := loadTLSCertificate(config.TLSCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	// Step 2: Create gRPC connection to the peer
	// This is the network connection that will be used for all communication
	grpcConn, err := grpc.Dial(
		config.PeerEndpoint,
		grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(tlsCert, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

// Step 3: Load the client identity (certificate and private key)
// This identity will be used to sign all transactions
id, sign, err := loadIdentity(config.MSPID, config.CertPath, config.KeyPath)
if err != nil {
grpcConn.Close()
return nil, fmt.Errorf("failed to load identity: %w", err)
}

// Step 4: Create the gateway connection to Fabric
// The gateway handles the transaction lifecycle: propose, endorse, submit, commit
gw, err := client.Connect(
id,
client.WithSign(sign),
client.WithClientConnection(grpcConn),
client.WithEvaluateTimeout(5*time.Second),
client.WithEndorseTimeout(15*time.Second),
client.WithSubmitTimeout(5*time.Second),
client.WithCommitStatusTimeout(1*time.Minute),
)
	if err != nil {
		grpcConn.Close()
		return nil, fmt.Errorf("failed to connect to gateway: %w", err)
	}

	// Step 5: Get the network (channel) and contract (chaincode) references
	network := gw.GetNetwork(config.ChannelName)
	contract := network.GetContract(config.ChaincodeName)

	return &FabricClient{
		gateway:       gw,
		network:       network,
		contract:      contract,
		grpcConn:      grpcConn,
		channelName:   config.ChannelName,
		chaincodeName: config.ChaincodeName,
	}, nil
}

// SubmitTransaction submits a transaction to the Fabric network
// This goes through the full transaction flow:
// 1. Propose to endorsing peers
// 2. Get endorsements
// 3. Submit to orderer
// 4. Wait for commit on ledger
func (fc *FabricClient) SubmitTransaction(functionName string, args ...string) ([]byte, error) {
	// Submit the transaction - this is a synchronous call that:
	// - Sends the proposal to endorsing peers
	// - Collects endorsements
	// - Submits the endorsed transaction to the orderer
	// - Waits for the transaction to be committed to the ledger
	result, err := fc.contract.SubmitTransaction(functionName, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to submit transaction: %w", err)
	}

	return result, nil
}

// EvaluateTransaction evaluates a transaction (query) without committing to ledger
// This is for read-only operations that don't change the world state
func (fc *FabricClient) EvaluateTransaction(functionName string, args ...string) ([]byte, error) {
	// Evaluate only queries the ledger, doesn't create a transaction
	result, err := fc.contract.EvaluateTransaction(functionName, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate transaction: %w", err)
	}

	return result, nil
}

// Close closes the gateway connection and gRPC connection
func (fc *FabricClient) Close() {
	if fc.gateway != nil {
		fc.gateway.Close()
	}
	if fc.grpcConn != nil {
		fc.grpcConn.Close()
	}
}

// loadTLSCertificate loads the peer's TLS certificate from file
func loadTLSCertificate(certPath string) (*x509.CertPool, error) {
	cert, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read TLS certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(cert) {
		return nil, fmt.Errorf("failed to add certificate to pool")
	}

	return certPool, nil
}

// loadIdentity loads the client identity (certificate and private key)
func loadIdentity(mspID, certPath, keyPath string) (*identity.X509Identity, identity.Sign, error) {
// Read the certificate
certPEM, err := os.ReadFile(certPath)
if err != nil {
return nil, nil, fmt.Errorf("failed to read certificate: %w", err)
}

// Parse the certificate
cert, err := identity.CertificateFromPEM(certPEM)
if err != nil {
return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
}

// Read the private key
keyPEM, err := os.ReadFile(keyPath)
if err != nil {
return nil, nil, fmt.Errorf("failed to read private key: %w", err)
}

// Parse the private key
key, err := identity.PrivateKeyFromPEM(keyPEM)
if err != nil {
return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
}

// Create a sign function using the private key
sign, err := identity.NewPrivateKeySign(key)
if err != nil {
return nil, nil, fmt.Errorf("failed to create sign function: %w", err)
}

// Create the identity
id, err := identity.NewX509Identity(mspID, cert)
if err != nil {
return nil, nil, fmt.Errorf("failed to create X509 identity: %w", err)
}

return id, sign, nil
}

// DefaultFabricConfig returns a default configuration for test-network
// This assumes you're using the standard Fabric test-network setup
func DefaultFabricConfig() FabricConfig {
	// Get the path to fabric-samples
	fabricSamplesPath := filepath.Join("/Users/sathvikcustiv/fabric-dev", "fabric-samples")
	testNetworkPath := filepath.Join(fabricSamplesPath, "test-network")
	
	return FabricConfig{
		// Org1's peer TLS certificate
		TLSCertPath: filepath.Join(testNetworkPath, 
			"organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt"),
		
		// Org1's peer endpoint
		PeerEndpoint: "localhost:7051",
		
		// Org1's MSP ID
		MSPID: "Org1MSP",
		
		// Admin user certificate for Org1
		CertPath: filepath.Join(testNetworkPath,
			"organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp/signcerts/cert.pem"),
		
		// Admin user private key for Org1
		KeyPath: filepath.Join(testNetworkPath,
			"organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp/keystore"),
		
		// Channel name (default for test-network)
		ChannelName: "mychannel",
		
		// Chaincode name (your priority wallet chaincode)
		ChaincodeName: "wallet",
	}
}

// GetPrivateKeyPath finds the first .pem or _sk file in a keystore directory
func GetPrivateKeyPath(keystorePath string) (string, error) {
	files, err := os.ReadDir(keystorePath)
	if err != nil {
		return "", fmt.Errorf("failed to read keystore directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			filename := file.Name()
			if filepath.Ext(filename) == ".pem" || 
			   len(filename) > 3 && filename[len(filename)-3:] == "_sk" {
				return filepath.Join(keystorePath, filename), nil
			}
		}
	}

	return "", fmt.Errorf("no private key found in keystore")
}
