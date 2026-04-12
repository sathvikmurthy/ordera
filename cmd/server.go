package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"priority-fabric-project/internal"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// parseWeights parses a comma-separated weight vector (e.g. "0.4,0.3,0.2,0.1")
// into a [4]float64 and validates that the weights sum to 1.0 (±0.001 tolerance).
func parseWeights(s string) ([4]float64, error) {
	var w [4]float64
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return w, fmt.Errorf("weights must have exactly 4 comma-separated values, got %d", len(parts))
	}
	sum := 0.0
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return w, fmt.Errorf("weight %d is not a number: %v", i, err)
		}
		if v < 0 {
			return w, fmt.Errorf("weight %d is negative: %v", i, v)
		}
		w[i] = v
		sum += v
	}
	if math.Abs(sum-1.0) > 0.001 {
		return w, fmt.Errorf("weights must sum to 1.0, got %.4f", sum)
	}
	return w, nil
}

func main() {
	var (
		port         = flag.String("port", "8080", "Server port")
		batchSize    = flag.Int("batch-size", 6, "Window Size (Block Size)")
		batchTimeout = flag.Duration("batch-timeout", 10*time.Second, "Max wait time before cutting a block")
		mempoolSize  = flag.Int("mempool-size", 1000, "Maximum transactions in the bucket system")
		useFabric    = flag.Bool("use-fabric", false, "Connect to Hyperledger Fabric")
		weightsStr   = flag.String("weights", "0.40,0.30,0.20,0.10",
			"WFQ weights per priority class (swap,borrow,lend,transfer). Must sum to 1.0")
	)
	flag.Parse()

	weights, err := parseWeights(*weightsStr)
	if err != nil {
		log.Fatalf("invalid --weights: %v", err)
	}

	fmt.Println("🚀 Priority Window Gateway: ACTIVE")
	fmt.Println("=====================================")
	fmt.Printf("📊 Strategy: Weighted Fair Queueing with Priority Spillover\n")
	fmt.Printf("⚖️  Weights:    swap=%.2f borrow=%.2f lend=%.2f transfer=%.2f\n",
		weights[0], weights[1], weights[2], weights[3])
	fmt.Printf("🪟 Window Size (Block Size): %d\n", *batchSize)
	fmt.Printf("📐 Quotas:     swap=%d borrow=%d lend=%d transfer=%d\n",
		int(weights[0]*float64(*batchSize)),
		int(weights[1]*float64(*batchSize)),
		int(weights[2]*float64(*batchSize)),
		int(weights[3]*float64(*batchSize)))
	fmt.Printf("⏱️  Heartbeat: %v\n", *batchTimeout)
	fmt.Printf("📈 Metrics: http://localhost:%s/metrics\n", *port)
	fmt.Println("=====================================")

	mempool := internal.NewMempool(*mempoolSize, weights)
	
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
	batcher := internal.NewBatcher(mempool, *batchSize, *batchTimeout, fabricClient)
	
	// 6. Initialize the HTTP Gateway
	gateway := internal.NewTransactionGateway(mempool, batcher)
	
	// 7. Start the Pulse
	log.Printf("🔄 Starting the batcher pulse...")
	go batcher.Start()
	
	// 8. Prometheus Metrics Endpoint
	// This registers the /metrics route on the default HTTP mux
	http.Handle("/metrics", promhttp.Handler())
	log.Printf("📊 Metrics exporter listening on /metrics")

	// 9. Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		log.Printf("🛑 Shutting down safely...")
		batcher.Stop()
		os.Exit(0)
	}()
	
	// 10. Start Server
	log.Printf("🌐 Gateway listening on port %s", *port)
	
	// Note: gateway.StartServer must use the default http.ListenAndServe 
	// or its own mux for the /metrics route to be visible.
	gateway.StartServer(*port)
}