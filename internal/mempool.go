package internal

import (
	"fmt"
	"sort"
	"sync"
	"priority-fabric-project/types"
)

type Mempool struct {
	// The Drawer System: [0]=Swap, [1]=Borrow, [2]=Lend, [3]=Transfer
	buckets [4][]*types.Transaction 
	mutex   sync.RWMutex
	maxSize int
	current int 
}

func NewMempool(maxSize int) *Mempool {
	return &Mempool{
		maxSize: maxSize,
	}
}

func (m *Mempool) AddTransaction(tx *types.Transaction) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.current >= m.maxSize {
		return fmt.Errorf("mempool is full")
	}

	// Transactions are appended, keeping each bucket naturally sorted by time (FIFO)
	m.buckets[tx.Priority] = append(m.buckets[tx.Priority], tx)
	m.current++
	return nil
}

// GetBatch captures the Window and decides to Sort (Priority) or stay Pure (FIFO)
func (m *Mempool) GetBatch(batchSize int, useWindowedSort bool) []*types.Transaction {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.current == 0 { return nil }

	// Phase 1: Capture the Window (Temporal Selection)
	// We find the oldest transactions across all buckets up to the batchSize
	var window []*types.Transaction
	limit := batchSize
	if m.current < limit { limit = m.current }

	for len(window) < limit {
		oldestIdx := -1

		// K-Way Merge: Compare the head of all 4 buckets to find the absolute oldest arrival
		for p := 0; p < 4; p++ {
			if len(m.buckets[p]) > 0 {
				if oldestIdx == -1 || m.buckets[p][0].Timestamp.Before(m.buckets[oldestIdx][0].Timestamp) {
					oldestIdx = p
				}
			}
		}

		if oldestIdx == -1 { break }

		// Move the transaction from its Bucket to the Window
		tx := m.buckets[oldestIdx][0]
		window = append(window, tx)
		m.buckets[oldestIdx] = m.buckets[oldestIdx][1:]
		m.current--
	}

	// Phase 2: Conditional Execution Order
	if useWindowedSort {
		// WINDOWED SORT: Priority (0->3) first, then Arrival Time
		sort.Slice(window, func(i, j int) bool {
			if window[i].Priority != window[j].Priority {
				return window[i].Priority < window[j].Priority
			}
			return window[i].Timestamp.Before(window[j].Timestamp)
		})
	}
	// Note: If useWindowedSort is false, it remains in Pure FIFO (Arrival Order)

	return window
}

// Requeue puts failed transactions back at the absolute front of their buckets
func (m *Mempool) Requeue(txs []*types.Transaction) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, tx := range txs {
		tx.Status = "pending"
		p := tx.Priority
		// Prepend to ensure seniority in the next available window
		m.buckets[p] = append([]*types.Transaction{tx}, m.buckets[p]...)
		m.current++
	}
}

func (m *Mempool) Size() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.current
}