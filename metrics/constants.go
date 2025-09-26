package metrics

import metricsLib "github.com/weeb-vip/go-metrics-lib"

// Metric result constants for easy usage
const (
	Success = metricsLib.Success
	Error   = metricsLib.Error
)

// Database method constants
const (
	MethodSelect = metricsLib.DatabaseMetricMethodSelect
	MethodInsert = metricsLib.DatabaseMetricMethodInsert
	MethodUpdate = metricsLib.DatabaseMetricMethodUpdate
	MethodDelete = metricsLib.DatabaseMetricMethodDelete
)

// Common table/component names
const (
	TableUsers        = "users"
	TableUserImages   = "user_images"

	ComponentResolver   = "resolver"
	ComponentService    = "service"
	ComponentRepository = "repository"
)