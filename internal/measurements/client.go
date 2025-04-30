package measurements

import (
	"log"

	"github.com/DataDog/datadog-go/statsd"
)

func New() Measurer {
	stats, err := statsd.
		New("",
			statsd.WithNamespace("backend"),
			statsd.WithTags([]string{"application:users-backend"}),
		)
	if err != nil {
		log.Default().Print("DogStatsD not found, ignoring...", err.Error())
	}

	return NewMeasurementClient(stats)
}
