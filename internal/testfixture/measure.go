package testfixture

import (
	"math"
	"slices"
	"testing"
	"time"
)

// Measure includes the operation, its result encoding and a small per-sample
// clock/bookkeeping cost. Fixture construction, sample allocation and percentile
// calculation are outside benchmark timing/allocation accounting.
func Measure(b *testing.B, call func() ([]byte, error)) {
	b.Helper()
	samples := make([]time.Duration, b.N)
	var totalBytes int64
	b.ReportAllocs()
	b.ResetTimer()
	for i := range b.N {
		start := time.Now()
		data, err := call()
		samples[i] = time.Since(start)
		if err != nil {
			b.Fatal(err)
		}
		totalBytes += int64(len(data))
	}
	b.StopTimer()
	b.ReportMetric(float64(totalBytes)/float64(b.N), "result-B/op")
	if len(samples) >= 20 {
		b.ReportMetric(float64(percentile(samples, .5))/float64(time.Millisecond), "p50-ms")
		b.ReportMetric(float64(percentile(samples, .95))/float64(time.Millisecond), "p95-ms")
	}
}

// Nearest-rank percentile; input is a measurement-owned slice, not user data.
func percentile(values []time.Duration, fraction float64) time.Duration {
	slices.Sort(values)
	return values[int(math.Ceil(float64(len(values))*fraction))-1]
}
