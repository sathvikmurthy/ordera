#!/bin/bash
# WFQ Quota Enforcement Demo
# Sends a heavy swap flood (90% swaps, 10% split among others) to demonstrate
# that WFQ enforces per-class quotas and transfer/lend do not starve even
# under adversarial swap-dominated load.
#
# Usage: ./wfq_demo.sh [workers] [duration]

SERVER="${SERVER:-http://localhost:8080}"
WORKERS="${1:-20}"
DURATION="${2:-0}"

# 90% swap, 5% borrow, 3% lend, 2% transfer — priority-skewed adversarial load
TX_TYPES=("swap" "swap" "swap" "swap" "swap" "swap" "swap" "swap" "swap" "swap"
          "swap" "swap" "swap" "swap" "swap" "swap" "swap" "swap"
          "borrow" "borrow" "borrow"
          "lend" "lend"
          "transfer")
USERS=("alice" "bob" "carol" "dave" "eve" "frank" "grace" "heidi")

START_TIME=$(date +%s)
TMP=$(mktemp -d)

echo "🔥 WFQ Quota Enforcement Demo"
echo "   Server  : $SERVER"
echo "   Workers : $WORKERS"
echo "   Load    : 75% swap, 12% borrow, 8% lend, 4% transfer (priority-skewed)"
echo "   Duration: $([ "$DURATION" -gt 0 ] && echo "${DURATION}s" || echo "unlimited")"
echo "   Ctrl+C to stop"
echo ""

cleanup() {
    kill 0 2>/dev/null
    TOTAL=0
    for f in "$TMP"/sent_*; do
        [ -f "$f" ] && { VAL=$(cat "$f" 2>/dev/null); TOTAL=$(( TOTAL + ${VAL:-0} )); }
    done
    ELAPSED=$(( $(date +%s) - START_TIME ))
    ELAPSED=$(( ELAPSED == 0 ? 1 : ELAPSED ))
    echo ""
    echo "🛑 Stopped"
    echo "   Total sent : $TOTAL"
    echo "   Avg TPS    : $(( TOTAL / ELAPSED ))"
    rm -rf "$TMP"
    exit 0
}
trap cleanup INT TERM

worker() {
    local id=$1
    local count=0
    echo 0 > "$TMP/sent_${id}"
    while true; do
        FROM="${USERS[$((RANDOM % ${#USERS[@]}))]}"
        TO="${USERS[$((RANDOM % ${#USERS[@]}))]}"
        AMOUNT="$((RANDOM % 990 + 10))"
        TXTYPE="${TX_TYPES[$((RANDOM % ${#TX_TYPES[@]}))]}"
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$SERVER/submit" \
            -H "Content-Type: application/json" \
            -d "{\"from\":\"$FROM\",\"to\":\"$TO\",\"amount\":\"$AMOUNT\",\"txType\":\"$TXTYPE\"}")
        if [ "$HTTP_CODE" = "200" ]; then
            count=$(( count + 1 ))
            echo "$count" > "$TMP/sent_${id}"
        fi
        if [ "$DURATION" -gt 0 ]; then
            ELAPSED=$(( $(date +%s) - START_TIME ))
            [ "$ELAPSED" -ge "$DURATION" ] && break
        fi
    done
}

for i in $(seq 1 "$WORKERS"); do
    worker "$i" &
done

while true; do
    ELAPSED=$(( $(date +%s) - START_TIME ))
    ELAPSED_D=$(( ELAPSED == 0 ? 1 : ELAPSED ))
    TOTAL=0
    for f in "$TMP"/sent_*; do
        [ -f "$f" ] && { VAL=$(cat "$f" 2>/dev/null); TOTAL=$(( TOTAL + ${VAL:-0} )); }
    done
    TPS=$(( TOTAL / ELAPSED_D ))
    MEMPOOL=$(curl -s "$SERVER/mempool/status" 2>/dev/null | grep -o '"total_mempool_size":[0-9]*' | grep -o '[0-9]*$')
    printf "\r  ⏱ %3ds | sent: %5d | tps: %4d | mempool: %4s   " \
        "$ELAPSED" "$TOTAL" "$TPS" "${MEMPOOL:-?}"
    if [ "$DURATION" -gt 0 ] && [ "$ELAPSED" -ge "$DURATION" ]; then
        cleanup
    fi
    sleep 1
done
