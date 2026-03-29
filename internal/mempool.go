package internal

import (
	"fmt"
	"sort"
	"sync"
	"priority-fabric-project/types"
)

type Mempool struct {
	buckets [4][]*types.Transaction 
	mutex   sync.RWMutex
	maxSize int
	current int 
}

func NewMempool(maxSize int) *Mempool {
	return &Mempool{maxSize: maxSize}
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

func (m *Mempool) GetBatch(batchSize int, useWindowedSort bool) []*types.Transaction {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.current == 0 { return nil }

	var window []*types.Transaction
	limit := batchSize
	if m.current < limit { limit = m.current }

	// Phase 1: Capture Window (K-Way Merge)
	for len(window) < limit {
		oldestIdx := -1
		for p := 0; p < 4; p++ {
			if len(m.buckets[p]) > 0 {
				if oldestIdx == -1 || m.buckets[p][0].Timestamp.Before(m.buckets[oldestIdx][0].Timestamp) {
					oldestIdx = p
				}
			}
		}
		if oldestIdx == -1 { break }

		tx := m.buckets[oldestIdx][0]
		window = append(window, tx)
		m.buckets[oldestIdx] = m.buckets[oldestIdx][1:]
		m.current--
	}

	// 📊 PROMETHEUS: Decrement gauge by the number of transactions removed
	MempoolSize.Sub(float64(len(window)))

	// Phase 2: Execution Sort
	if useWindowedSort {
		sort.Slice(window, func(i, j int) bool {
			if window[i].Priority != window[j].Priority {
				return window[i].Priority < window[j].Priority
			}
			return window[i].Timestamp.Before(window[j].Timestamp)
		})
	}

	return window
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