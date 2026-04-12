#!/bin/bash
# Continuous random transaction load generator
# Sends a steady stream of mixed-priority transactions to visualize on Grafana

SERVER="${1:-http://localhost:8080}"
DELAY="${2:-0.3}"  # seconds between transactions (default 0.3s ≈ ~3 tx/sec)

USERS=("alice" "bob" "carol" "dave" "eve" "frank" "grace" "heidi" "ivan" "judy")
TX_TYPES=("swap" "swap" "swap" "borrow" "borrow" "lend" "transfer")  # weighted: more swaps
COUNT=0

echo "🚀 Load generator started"
echo "   Server : $SERVER"
echo "   Delay  : ${DELAY}s between transactions"
echo "   Press Ctrl+C to stop"
echo ""

submit() {
    local from="${USERS[$((RANDOM % ${#USERS[@]}))]}"
    local to="${USERS[$((RANDOM % ${#USERS[@]}))]}"
    local amount="$((RANDOM % 990 + 10))"
    local txtype="${TX_TYPES[$((RANDOM % ${#TX_TYPES[@]}))]}"

    local result
    result=$(curl -s -X POST "$SERVER/submit" \
        -H "Content-Type: application/json" \
        -d "{\"from\":\"$from\",\"to\":\"$to\",\"amount\":\"$amount\",\"txType\":\"$txtype\"}")

    local priority
    priority=$(echo "$result" | grep -o '"priority":[0-9]' | grep -o '[0-9]')
    COUNT=$((COUNT + 1))
    printf "  [%4d] %-8s → %-8s  %4s  type=%-8s  priority=%s\n" \
        "$COUNT" "$from" "$to" "$amount" "$txtype" "$priority"
}

trap 'echo ""; echo "🛑 Stopped after $COUNT transactions."; exit 0' INT

while true; do
    submit
    sleep "$DELAY"
done
