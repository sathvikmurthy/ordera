package internal

import (
	"log"
	"sync"
	"time"
	"priority-fabric-project/types"
)

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

// BatchProcessor handles the actual submission to the ledger
type BatchProcessor interface {
	ProcessBatch(batch []*types.Transaction) error
}

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
			if b.mempool.Size() > 0 {
				b.processBatch("timeout")
			}
		default:
			// Trigger block creation when mempool reaches the Block Size
			if b.mempool.Size() >= b.batchSize {
				b.processBatch("size-limit")
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (b *Batcher) processBatch(trigger string) {
	b.batchCount++
	
	// Alternate Logic: Odd Blocks = Windowed Sort, Even Blocks = Pure FIFO
	useWindowedSort := (b.batchCount % 2) != 0
	
	batch := b.mempool.GetBatch(b.batchSize, useWindowedSort)
	if len(batch) == 0 { return }

	modeLabel := "PURE FIFO"
	if useWindowedSort { modeLabel = "WINDOWED SORT" }
	
	log.Printf("⚡ Block #%d [%s] Trigger: %s, Size: %d", b.batchCount, modeLabel, trigger, len(batch))

	// Attempt to process the batch
	err := b.processor.ProcessBatch(batch)
	if err != nil {
		log.Printf("❌ Block #%d failed: %v. Re-queuing transactions...", b.batchCount, err)
		b.mempool.Requeue(batch)
	} else {
		b.mutex.Lock()
		b.totalProcessed += len(batch)
		b.mutex.Unlock()
		log.Printf("✅ Block #%d committed. Total processed: %d", b.batchCount, b.totalProcessed)
	}
}

func (b *Batcher) Stop() { b.stopChan <- true }