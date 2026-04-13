package internal

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"priority-fabric-project/types"
)

type Mempool struct {
	buckets [4][]*types.Transaction
	mutex   sync.RWMutex
	maxSize int
	current int
	weights [4]float64 // WFQ weights per priority class (must sum to 1.0)
}

// NewMempool creates a mempool with the given capacity and WFQ weight vector.
// weights[p] is the fraction of each block reserved for priority class p.
func NewMempool(maxSize int, weights [4]float64) *Mempool {
	return &Mempool{maxSize: maxSize, weights: weights}
}

func (m *Mempool) AddTransaction(tx *types.Transaction) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.current >= m.maxSize {
		return fmt.Errorf("mempool is full")
	}

	m.buckets[tx.Priority] = append(m.buckets[tx.Priority], tx)
	m.current++

	// 📊 PROMETHEUS: Update the gauge
	MempoolSize.Inc() 

	return nil
}

// GetBatch builds a block using Weighted Fair Queueing with Priority Spillover.
//
//	Phase 1 — Quota Allocation:
//	    each priority class p contributes up to floor(weights[p] * batchSize)
//	    transactions, popped from the head of B_p (FIFO within class).
//
//	Phase 2 — Priority Spillover:
//	    any remaining slots are filled from the highest-priority non-empty bucket,
//	    cascading downward. This guarantees that unused quota flows to the most
//	    urgent transactions first.
//
// Bounded-latency guarantee: a transaction at position i within bucket p waits
// at most ceil(i / quota_p) blocks before inclusion.
func (m *Mempool) GetBatch(batchSize int) []*types.Transaction {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.current == 0 {
		return nil
	}

	block := make([]*types.Transaction, 0, batchSize)

	// Phase 1 — Quota Allocation
	for p := 0; p < 4; p++ {
		quota := int(math.Floor(m.weights[p] * float64(batchSize)))
		taken := 0
		for taken < quota && len(m.buckets[p]) > 0 && len(block) < batchSize {
			block = append(block, m.buckets[p][0])
			m.buckets[p] = m.buckets[p][1:]
			m.current--
			taken++
		}
	}

	// Phase 2 — Priority Spillover
	for len(block) < batchSize {
		highest := -1
		for p := 0; p < 4; p++ {
			if len(m.buckets[p]) > 0 {
				highest = p
				break
			}
		}
		if highest == -1 {
			break
		}
		block = append(block, m.buckets[highest][0])
		m.buckets[highest] = m.buckets[highest][1:]
		m.current--
	}

	// Phase 3 — Intra-Block Priority Sort (stable).
	// Guarantees strict priority order within the committed block so that no
	// lower-priority transaction is submitted to Fabric before a higher-priority
	// one in the same block. FIFO ordering within a priority class is preserved
	// via the stable sort falling back to arrival timestamp.
	sort.SliceStable(block, func(i, j int) bool {
		if block[i].Priority != block[j].Priority {
			return block[i].Priority < block[j].Priority
		}
		return block[i].Timestamp.Before(block[j].Timestamp)
	})

	// Record gateway-side wait time (submit → extraction) for each extracted tx
	now := time.Now()
	for _, tx := range block {
		WaitLatency.WithLabelValues(tx.TxType).Observe(now.Sub(tx.Timestamp).Seconds())
	}

	MempoolSize.Sub(float64(len(block)))
	return block
}

func (m *Mempool) Requeue(txs []*types.Transaction) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, tx := range txs {
		tx.Status = "pending"
		p := tx.Priority
		m.buckets[p] = append([]*types.Transaction{tx}, m.buckets[p]...)
		m.current++
		
		// 📊 PROMETHEUS: Put them back in the gauge
		MempoolSize.Inc()
	}
}

func (m *Mempool) Size() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.current
}

// Utilization returns the fraction of maxSize currently used, in range [0.0, 1.0].
func (m *Mempool) Utilization() float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if m.maxSize == 0 {
		return 0
	}
	return float64(m.current) / float64(m.maxSize)
}