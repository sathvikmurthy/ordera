#!/bin/bash

# Test script for quota-based anti-starvation algorithm
# Demonstrates that lower priority transactions are not starved

echo "🧪 Testing Quota-Based Anti-Starvation Algorithm"
echo "=================================================="
echo ""
echo "This test will submit transactions matching your example:"
echo "Mempool: [0, 0, 0, 0, 0, 0, 0, 1, 1, 2, 3]"
echo ""
echo "Expected batches with window size 5:"
echo "Batch 1: [0, 0, 0, 1, 2]  ← Mix of all priorities"
echo "Batch 2: [0, 0, 0, 0, 1]"
echo "Batch 3: [0, 0, 3]"
echo ""

SERVER="http://localhost:8081"

# Function to submit a transaction
submit_tx() {
    local from=$1
    local to=$2
    local amount=$3
    local txtype=$4
    
    curl -s -X POST $SERVER/submit \
        -H "Content-Type: application/json" \
        -d "{\"from\":\"$from\",\"to\":\"$to\",\"amount\":\"$amount\",\"txType\":\"$txtype\"}" \
        > /dev/null
}

echo "📤 Submitting 11 transactions (7 swap, 2 borrow, 1 lend, 1 transfer)..."
echo ""

# Submit 7 swap transactions (priority 0)
for i in {1..7}; do
    submit_tx "user${i}" "userA" "$((i*10))" "swap"
    echo "  ✓ Submitted swap transaction $i"
done

# Submit 2 borrow transactions (priority 1)
for i in {1..2}; do
    submit_tx "borrower${i}" "userB" "$((i*20))" "borrow"
    echo "  ✓ Submitted borrow transaction $i"
done

# Submit 1 lend transaction (priority 2)
submit_tx "lender1" "userC" "50" "lend"
echo "  ✓ Submitted lend transaction"

# Submit 1 transfer transaction (priority 3)
submit_tx "sender1" "userD" "30" "transfer"
echo "  ✓ Submitted transfer transaction"

echo ""
echo "✅ All 11 transactions submitted!"
echo ""
echo "📊 Current mempool status:"
curl -s $SERVER/mempool/status | jq '{
    total: .stats.totalTransactions,
    byPriority: .stats.byPriority
}'

echo ""
echo "⏳ Waiting for batches to be processed..."
echo "   (Watch the server logs to see the quota-based batching in action)"
echo ""
echo "📝 Expected behavior:"
echo "   - Each batch should include at least 1 tx from each priority (if available)"
echo "   - Remaining slots filled with highest priority (swap)"
echo "   - Low priority transactions (transfer) won't be starved"
echo ""
echo "💡 Check server terminal for detailed batch composition logs!"
