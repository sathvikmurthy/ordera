package internal

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 1. Current Mempool Depth
	MempoolSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gateway_mempool_size",
		Help: "Current transactions in the bucket system",
	})

	// 2. Throughput Counter (Labeled by Block Mode)
	TxProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_tx_processed_total",
		Help: "Total transactions committed to Fabric",
	}, []string{"mode"})

	// 3. Latency Histogram (Critical for HFT research)
	BlockCreationLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_block_creation_seconds",
		Help:    "Time taken to merge buckets and sort the window",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1}, // Millisecond precision
	}, []string{"mode"})

	// 4. Gas Fee Gauge (labeled by transaction type)
	GasFeeGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gateway_gas_fee",
		Help: "Most recently calculated dynamic gas fee by transaction type",
	}, []string{"tx_type"})

	// 5. Treasury Balance — cumulative gas fees collected since gateway start
	TreasuryCollected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_treasury_collected_total",
		Help: "Total gas fees collected by the network treasury since gateway start",
	})

	// 6. Block Composition — per-priority-class throughput for validating WFQ quotas
	BlockComposition = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_block_composition_total",
		Help: "Total transactions committed to blocks, labeled by transaction type",
	}, []string{"tx_type"})

	// 7. End-to-End Latency — submit → on-chain commit, per tx_type
	E2ELatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_tx_e2e_seconds",
		Help:    "End-to-end latency from client submit to on-chain commit, by transaction type",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 16),
	}, []string{"tx_type"})

	// 8. Wait Latency — submit → extraction from mempool (gateway queueing delay only)
	WaitLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_tx_wait_seconds",
		Help:    "Gateway wait time from submit to extraction from mempool, by transaction type",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 16),
	}, []string{"tx_type"})

	// 9. Block Position — where each tx ends up within its committed block.
	// After the Phase 3 priority sort, swap should always land at low positions
	// and transfer at high positions. Use _sum/_count for avg, histogram_quantile
	// for p50/p95/p99 position per tx type.
	BlockPosition = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gateway_tx_block_position",
		Help:    "Position of transaction within its committed block (1 = first, N = last), by type",
		Buckets: prometheus.LinearBuckets(1, 1, 30),
	}, []string{"tx_type"})

	// 10. Last Block Slot (COMMITTED ORDER) — snapshot of the most recently
	// committed block in the order transactions were submitted to Fabric (i.e.
	// after WFQ + Phase 3 priority sort).
	LastBlockSlot = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gateway_last_block_slot",
		Help: "Most recently committed block in committed (priority-sorted) order",
	}, []string{"position", "tx_type"})

	// 11. Last Block Arrival Slot (ARRIVAL ORDER) — same block as LastBlockSlot,
	// but reordered by arrival timestamp. Used side-by-side with LastBlockSlot in
	// Grafana to visualize the before/after of WFQ + priority sort.
	LastBlockArrivalSlot = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gateway_last_block_arrival_slot",
		Help: "Most recently committed block in arrival-timestamp order (pre-sort input)",
	}, []string{"position", "tx_type"})
)