package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "priority-fabric-project/internal"
)

func main() {
    // Command line flags
    var (
        port         = flag.String("port", "8080", "Server port")
        batchSize    = flag.Int("batch-size", 5, "Number of transactions per batch")
        batchTimeout = flag.Duration("batch-timeout", 30*time.Second, "Batch timeout duration")
        mempoolSize  = flag.Int("mempool-size", 1000, "Maximum transactions in mempool")
        verbose      = flag.Bool("verbose", false, "Enable verbose logging")
        useFabric    = flag.Bool("use-fabric", false, "Connect to Fabric network (requires network to be running)")
    )
    flag.Parse()
    
    // Print startup banner
    fmt.Println("🚀 Priority Fabric Transaction Gateway")
    fmt.Println("=====================================")
    fmt.Printf("📊 Configuration:\n")
    fmt.Printf("   Port: %s\n", *port)
    fmt.Printf("   Batch Size: %d transactions\n", *batchSize)
    fmt.Printf("   Batch Timeout: %v\n", *batchTimeout)
    fmt.Printf("   Mempool Size: %d transactions\n", *mempoolSize)
    fmt.Printf("   Verbose Logging: %t\n", *verbose)
    fmt.Printf("   Fabric Integration: %t\n", *useFabric)
    fmt.Println()
    
    // Set log level
    if !*verbose {
        log.SetFlags(log.LstdFlags | log.Lshortfile)
    }
    
    // Validate configuration
    if *batchSize <= 0 {
        log.Fatal("❌ Batch size must be greater than 0")
    }
    if *mempoolSize <= 0 {
        log.Fatal("❌ Mempool size must be greater than 0")
    }
    if *batchTimeout <= 0 {
        log.Fatal("❌ Batch timeout must be greater than 0")
    }
    
    log.Printf("🔧 Initializing components...")
    
    // Create mempool
    mempool := internal.NewMempool(*mempoolSize)
    log.Printf("✅ Mempool initialized (max size: %d)", *mempoolSize)
    
    // Initialize Fabric client if requested
    var fabricClient *internal.FabricClient
    
    if *useFabric {
        log.Printf("🔗 Connecting to Hyperledger Fabric network...")
        
        // Get default configuration for test-network
        config := internal.DefaultFabricConfig()
        
        // Find the private key in the keystore directory
        keyPath, err := internal.GetPrivateKeyPath(config.KeyPath)
        if err != nil {
            log.Printf("⚠️  Warning: Could not find private key: %v", err)
            log.Printf("⚠️  Continuing in simulation mode without Fabric connection")
        } else {
            config.KeyPath = keyPath
            
            // Create Fabric client
            fabricClient, err = internal.NewFabricClient(config)
            if err != nil {
                log.Printf("⚠️  Warning: Failed to connect to Fabric: %v", err)
                log.Printf("⚠️  Continuing in simulation mode without Fabric connection")
                fabricClient = nil
            } else {
                log.Printf("✅ Connected to Fabric network!")
                log.Printf("   Channel: %s", config.ChannelName)
                log.Printf("   Chaincode: %s", config.ChaincodeName)
                log.Printf("   Organization: %s", config.MSPID)
                
                // Ensure client cleanup on shutdown
                defer fabricClient.Close()
            }
        }
    } else {
        log.Printf("ℹ️  Running in simulation mode (use --use-fabric to connect to Fabric network)")
    }
    
    // Create batcher with optional Fabric client
    batcher := internal.NewBatcher(mempool, *batchSize, *batchTimeout, fabricClient)
    log.Printf("✅ Batcher initialized (size: %d, timeout: %v)", *batchSize, *batchTimeout)
    
    // Create gateway
    gateway := internal.NewTransactionGateway(mempool, batcher)
    log.Printf("✅ Transaction gateway initialized")
    
    // Start batcher in background
    log.Printf("🔄 Starting batcher...")
    go batcher.Start()
    
    // Handle graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        log.Printf("🛑 Received shutdown signal, stopping batcher...")
        batcher.Stop()
        log.Printf("👋 Goodbye!")
        os.Exit(0)
    }()
    
    // Print priority information
    fmt.Printf("💎 Transaction Priority System:\n")
    fmt.Printf("   0. swap     - Highest priority (processed first)\n")
    fmt.Printf("   1. borrow   - High priority\n")
    fmt.Printf("   2. lend     - Medium priority\n")
    fmt.Printf("   3. transfer - Lowest priority (processed last)\n")
    fmt.Println()
    
    // Start HTTP server (blocking)
    log.Printf("🌐 Starting HTTP server...")
    gateway.StartServer(*port)
}

// Add some helper functions for demonstration
func init() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    
    // Print ASCII art banner
    banner := `
  ____       _            _ _         
 |  _ \ _ __(_) ___  _ __(_) |_ _   _ 
 | |_) | '__| |/ _ \| '__| | __| | | |
 |  __/| |  | | (_) | |  | | |_| |_| |
 |_|   |_|  |_|\___/|_|  |_|\__|\__, |
  _____     _          _        |___/ 
 |  ___|_ _| |__  _ __(_) ___          
 | |_ / _` + "`" + ` | '_ \| '__| |/ __|         
 |  _| (_| | |_) | |  | | (__          
 |_|  \__,_|_.__/|_|  |_|\___|         
   ____       _                       
  / ___| __ _| |_ _____      ____ _ _  
 | |  _ / _` + "`" + ` | __|/ _ \ \ /\ / / _` + "`" + ` | | 
 | |_| | (_| | |_|  __/\ V  V / (_| | |
  \____|\__,_|\__|\___|_\_/\_/ \__,_|_|
    `
    
    fmt.Println(banner)
}
