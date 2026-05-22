package metrics

import "github.com/prometheus/client_golang/prometheus"

var TelemetryReceivedTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "telemetry_received_total",
		Help: "Total number of telemetry events received.",
	},
	[]string{"asset_id", "status"},
)

var AssetTemperatureCelsius = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "asset_temperature_celsius",
		Help: "Latest asset temperature in Celsius.",
	},
	[]string{"asset_id"},
)

var AssetCPUUsagePercent = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "asset_cpu_usage_percent",
		Help: "Latest asset CPU usage percent.",
	},
	[]string{"asset_id"},
)

var AssetMemoryUsagePercent = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "asset_memory_usage_percent",
		Help: "Latest asset memory usage percent.",
	},
	[]string{"asset_id"},
)

var AssetStatus = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "asset_status",
		Help: "Latest asset status. Label status is set to 1 for current status and 0 for other known statuses.",
	},
	[]string{"asset_id", "status"},
)

func Register() {
	prometheus.MustRegister(TelemetryReceivedTotal)
	prometheus.MustRegister(AssetTemperatureCelsius)
	prometheus.MustRegister(AssetCPUUsagePercent)
	prometheus.MustRegister(AssetMemoryUsagePercent)
	prometheus.MustRegister(AssetStatus)
}
