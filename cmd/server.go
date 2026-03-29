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
	// 1. Command line flags - This is where your "Window Size" is set!
	var (
		port         = flag.String("port", "8080", "Server port")
		batchSize    = flag.Int("batch-size", 6, "Window Size (Block Size)") // Updated default to your example 6
		batchTimeout = flag.Duration("batch-timeout", 10*time.Second, "Max wait time before cutting a block")
		mempoolSize  = flag.Int("mempool-size", 1000, "Maximum transactions in the bucket system")
		verbose      = flag.Bool("verbose", true, "Enable detailed logging of window merges")
		useFabric    = flag.Bool("use-fabric", false, "Connect to Hyperledger Fabric")
	)
	flag.Parse()
	
	// 2. The Startup Banner (Updated for your Research Paper)
	fmt.Println("🚀 Priority Window Gateway: ACTIVE")
	fmt.Println("=====================================")
	fmt.Printf("📊 Strategy: Alternating [Windowed Sort] & [Pure FIFO]\n")
	fmt.Printf("🪟 Window Size (Block Size): %d\n", *batchSize)
	fmt.Printf("⏱️  Heartbeat: %v\n", *batchTimeout)
	fmt.Println("=====================================")
	
	// 3. Initialize the "Drawer System" (Mempool)
	// This uses the 4-bucket strategy we discussed
	mempool := internal.NewMempool(*mempoolSize)
	
	// 4. Initialize Fabric Client (Optional)
	var fabricClient *internal.FabricClient
	if *useFabric {
		config := internal.DefaultFabricConfig()
		keyPath, err := internal.GetPrivateKeyPath(config.KeyPath)
		if err == nil {
			config.KeyPath = keyPath
			fabricClient, _ = internal.NewFabricClient(config)
		}
	}

	// 5. Initialize the Batcher
	// This will now automatically alternate modes using b.batchCount
	batcher := internal.NewBatcher(mempool, *batchSize, *batchTimeout, fabricClient)
	
	// 6. Initialize the HTTP Gateway
	// Note: Ensure your internal/gateway.go is updated to match these components
	gateway := internal.NewTransactionGateway(mempool, batcher)
	
	// 7. Start the Pulse
	log.Printf("🔄 Starting the batcher pulse...")
	go batcher.Start()
	
	// 8. Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		log.Printf("🛑 Shutting down safely...")
		batcher.Stop()
		os.Exit(0)
	}()
	
	// 9. Start Server
	log.Printf("🌐 Gateway listening on port %s", *port)
	gateway.StartServer(*port)
}