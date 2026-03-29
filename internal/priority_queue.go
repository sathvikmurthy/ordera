package internal

import (
    "fmt"
    "priority-fabric-project/types"
)

// PriorityQueue implements heap.Interface for Transaction
type PriorityQueue []*types.Transaction

func (pq PriorityQueue) Len() int { 
    return len(pq) 
}

func (pq PriorityQueue) Less(i, j int) bool {
    // Lower priority number = higher priority (0 is highest)
    if pq[i].Priority != pq[j].Priority {
        return pq[i].Priority < pq[j].Priority
    }
    
    // If same priority, older transactions come first (FIFO within priority)
    return pq[i].Timestamp.Before(pq[j].Timestamp)
}

func (pq PriorityQueue) Swap(i, j int) {
    pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
    *pq = append(*pq, x.(*types.Transaction))
}

func (pq *PriorityQueue) Pop() interface{} {
    old := *pq
    n := len(old)
    if n == 0 {
        return nil
    }
    
    item := old[n-1]
    *pq = old[0 : n-1]
    return item
}

// Peek returns the highest priority transaction without removing it
func (pq *PriorityQueue) Peek() *types.Transaction {
    if len(*pq) == 0 {
        return nil
    }
    return (*pq)[0]
}

// String provides a string representation of the priority queue
func (pq PriorityQueue) String() string {
    if len(pq) == 0 {
        return "PriorityQueue: empty"
    }
    
    result := "PriorityQueue:\n"
    for i, tx := range pq {
        result += fmt.Sprintf("  [%d] ID: %s, Type: %s, Priority: %d, From: %s\n", 
            i, tx.ID, tx.TxType, tx.Priority, tx.From)
    }
    return result
}