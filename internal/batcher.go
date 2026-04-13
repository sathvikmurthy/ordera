package internal

import (
	"fmt"
	"log"
	"sort"
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

// processBatch builds a block via WFQ and submits it to Fabric.
// Every block uses the same weighted fair queueing algorithm — no mode alternation.
func (b *Batcher) processBatch(trigger string) {
	b.batchCount++

	const mode = "wfq"

	start := time.Now()
	batch := b.mempool.GetBatch(b.batchSize)
	BlockCreationLatency.WithLabelValues(mode).Observe(time.Since(start).Seconds())

	if len(batch) == 0 {
		return
	}

	// Count per-type composition of this block (for quota validation in Grafana)
	composition := map[string]int{}
	for _, tx := range batch {
		composition[tx.TxType]++
	}

	log.Printf("⚡ Block #%d [WFQ] Trigger: %s, Size: %d, Composition: %v",
		b.batchCount, trigger, len(batch), composition)

	err := b.processor.ProcessBatch(batch)
	if err != nil {
		log.Printf("❌ Block #%d failed (MVCC/Network): %v. Re-queuing...", b.batchCount, err)
		b.mempool.Requeue(batch)
		return
	}

	b.mutex.Lock()
	b.totalProcessed += len(batch)
	b.mutex.Unlock()

	TxProcessed.WithLabelValues(mode).Add(float64(len(batch)))
	commitTime := time.Now()

	// Snapshot the last committed block — COMMITTED ORDER (after Phase 3 sort)
	LastBlockSlot.Reset()
	for i, tx := range batch {
		BlockComposition.WithLabelValues(tx.TxType).Inc()
		E2ELatency.WithLabelValues(tx.TxType).Observe(commitTime.Sub(tx.Timestamp).Seconds())
		BlockPosition.WithLabelValues(tx.TxType).Observe(float64(i + 1))

		LastBlockSlot.WithLabelValues(
			fmt.Sprintf("%02d", i+1),
			tx.TxType,
		).Set(float64(tx.Priority))
	}

	// Snapshot the SAME block in ARRIVAL ORDER (sorted by submit timestamp).
	// This is the "before" image for the before/after comparison in Grafana.
	arrivalOrder := make([]*types.Transaction, len(batch))
	copy(arrivalOrder, batch)
	sort.SliceStable(arrivalOrder, func(i, j int) bool {
		return arrivalOrder[i].Timestamp.Before(arrivalOrder[j].Timestamp)
	})
	LastBlockArrivalSlot.Reset()
	for i, tx := range arrivalOrder {
		LastBlockArrivalSlot.WithLabelValues(
			fmt.Sprintf("%02d", i+1),
			tx.TxType,
		).Set(float64(tx.Priority))
	}

	log.Printf("✅ Block #%d committed. Total processed: %d", b.batchCount, b.totalProcessed)
}

func (b *Batcher) Stop() {
	b.stopChan <- true
}