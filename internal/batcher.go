package internal

import (
	"log"
	"sync"
	"time"
	"priority-fabric-project/types"
)

// BatchProcessor defines the interface for submitting batches to Fabric
type BatchProcessor interface {
	ProcessBatch(batch []*types.Transaction) error
}

// FabricBatchProcessor is the concrete implementation that uses the FabricClient
type FabricBatchProcessor struct {
	fabricClient *FabricClient
}

func (f *FabricBatchProcessor) ProcessBatch(batch []*types.Transaction) error {
	if f.fabricClient == nil {
		// Simulation mode if no Fabric connection is active [cite: 4]
		return nil
	}
	
	// Convert batch to the format required by your chaincode [cite: 1]
	for _, tx := range batch {
		_, err := f.fabricClient.SubmitTransaction("Transact", tx.From, tx.To, tx.Amount, tx.TxType, tx.GasFee)
		if err != nil {
			return err
		}
	}
	return nil
}

type Batcher struct {
	mempool        *Mempool
	batchSize      int
	batchTimeout   time.Duration
	processor      BatchProcessor
	stopChan       chan bool
	running        bool
	batchCount     int
	totalProcessed int
	mutex          sync.RWMutex
}

// NewBatcher initializes the batcher with the Window Size and Timeout
func NewBatcher(mempool *Mempool, size int, timeout time.Duration, client *FabricClient) *Batcher {
	return &Batcher{
		mempool:      mempool,
		batchSize:    size,
		batchTimeout: timeout,
		processor: &FabricBatchProcessor{
			fabricClient: client,
		},
		stopChan: make(chan bool),
	}
}

// Start kicks off the background pulse for block creation
func (b *Batcher) Start() {
	b.running = true
	ticker := time.NewTicker(b.batchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-b.stopChan:
			b.running = false
			return
		case <-ticker.C:
			// Trigger block if time expires and mempool isn't empty
			if b.mempool.Size() > 0 {
				b.processBatch("timeout")
			}
		default:
			// Trigger block as soon as the Window (Block Size) is full
			if b.mempool.Size() >= b.batchSize {
				b.processBatch("size-limit")
			}
			// Small sleep to prevent 100% CPU usage on your i7
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// processBatch handles the core logic of capturing the window and sorting
func (b *Batcher) processBatch(trigger string) {
	b.batchCount++
	
	// Logic to alternate modes: Odd blocks use Windowed Sort, Even use Pure FIFO
	useWindowedSort := (b.batchCount % 2) != 0
	mode := "pure_fifo"
	if useWindowedSort {
		mode = "windowed_sort"
	}

	// ⏱️ Start Timer for Prometheus
	start := time.Now()

	// Capture the window from the bucket system
	batch := b.mempool.GetBatch(b.batchSize, useWindowedSort)
	
	// Record the internal processing duration (merge + sort)
	BlockCreationLatency.WithLabelValues(mode).Observe(time.Since(start).Seconds())

	if len(batch) == 0 {
		return
	}

	modeLabel := "PURE FIFO"
	if useWindowedSort {
		modeLabel = "WINDOWED SORT"
	}
	
	log.Printf("⚡ Block #%d [%s] Trigger: %s, Size: %d", b.batchCount, modeLabel, trigger, len(batch))

	// Attempt to process the batch through the Fabric Client
	err := b.processor.ProcessBatch(batch)
	if err != nil {
		log.Printf("❌ Block #%d failed (MVCC/Network): %v. Re-queuing...", b.batchCount, err)
		b.mempool.Requeue(batch) // Preserve seniority at the front of buckets
	} else {
		b.mutex.Lock()
		b.totalProcessed += len(batch)
		b.mutex.Unlock()

		// 📊 Increment throughput counter for Grafana
		TxProcessed.WithLabelValues(mode).Add(float64(len(batch)))

		log.Printf("✅ Block #%d committed. Total processed: %d", b.batchCount, b.totalProcessed)
	}
}

func (b *Batcher) Stop() {
	b.stopChan <- true
}