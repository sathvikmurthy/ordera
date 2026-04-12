package internal

import "math"

// priorityMultipliers maps transaction type to its fee multiplier.
var priorityMultipliers = map[string]float64{
	"swap":     4.0,
	"borrow":   3.0,
	"lend":     2.0,
	"transfer": 1.0,
}

// CalculateGasFee computes the dynamic gas fee using a sigmoid congestion curve.
//
//	Gas Fee = 0.001 × PriorityMultiplier × CongestionFactor
//	CongestionFactor = 1 + (10.0 / (1 + e^(-8 × (u - 0.5))))
//	u = utilization (current mempool size / max mempool size), clamped to [0, 1]
//
// Congestion factor range: ~1.18x at 0% load → 10.82x at 100% load (hard ceiling: 11x)
func CalculateGasFee(txType string, utilization float64) float64 {
	multiplier, ok := priorityMultipliers[txType]
	if !ok {
		multiplier = 1.0
	}

	if utilization < 0 {
		utilization = 0
	} else if utilization > 1 {
		utilization = 1
	}

	congestionFactor := 1.0 + (10.0 / (1.0 + math.Exp(-8.0*(utilization-0.5))))
	return 0.001 * multiplier * congestionFactor
}
