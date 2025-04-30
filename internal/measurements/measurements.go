package measurements

import "github.com/DataDog/datadog-go/statsd"

// Measurer will be how we'll be sending statsd data everywhere.
type Measurer interface {
	// MeasureExecutionTime when deferred will measure and send execution time of that method body.
	MeasureExecutionTime(label string, tags []string) func()
	MarkEvent(label string, tags []string)
	MarkEventWithCount(label string, count int, tags []string)
}

type measurement struct {
	statsd statsd.ClientInterface
}

func NewMeasurementClient(client statsd.ClientInterface) Measurer {
	return measurement{statsd: client}
}
