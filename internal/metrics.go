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
)