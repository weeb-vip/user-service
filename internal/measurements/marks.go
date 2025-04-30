package measurements

func (m measurement) MarkEvent(label string, tags []string) {
	_ = m.statsd.Count(label, 1, tags, 1)
}

func (m measurement) MarkEventWithCount(label string, value int, tags []string) {
	_ = m.statsd.Count(label, int64(value), tags, 1)
}
