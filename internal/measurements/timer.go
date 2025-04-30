package measurements

import "time"

// MeasureExecutionTime will measure execution time of a function when defer'ed.
func (m measurement) MeasureExecutionTime(label string, tags []string) func() {
	now := time.Now()

	return func() {
		elapsed := time.Since(now)
		_ = m.statsd.Timing(label, elapsed, tags, 1)
	}
}
