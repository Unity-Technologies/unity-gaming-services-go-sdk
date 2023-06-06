package sqp

import (
	"math"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server/proto"
)

type (
	// sqpMetricsInfo holds the server metrics chunk data.
	sqpMetricsInfo struct {
		Count  byte
		Values []float32
	}
)

// MaxMetrics represents the maximum number of metrics one SQP packet supports.
const MaxMetrics = 10

// queryStateToMetrics converts metrics data in provided query state to sqpMetricsInfo.
func queryStateToMetrics(qs *proto.QueryState) *sqpMetricsInfo {
	l := byte(math.Min(float64(len(qs.Metrics)), float64(MaxMetrics)))
	m := &sqpMetricsInfo{
		Count: l,
	}

	m.Values = make([]float32, l)
	copy(m.Values, qs.Metrics)

	return m
}

// Size returns the number of bytes QueryResponder will use on the wire.
func (m sqpMetricsInfo) Size() uint32 {
	return uint32(1 + len(m.Values)*4)
}
