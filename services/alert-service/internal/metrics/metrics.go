package metrics

import "github.com/prometheus/client_golang/prometheus"

var AlertsCreatedTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "alerts_created_total",
		Help: "Total number of alerts created.",
	},
	[]string{"asset_id", "name", "severity"},
)

var AlertsResolvedTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "alerts_resolved_total",
		Help: "Total number of alerts resolved.",
	},
	[]string{"asset_id", "name", "severity"},
)

func Register() {
	prometheus.MustRegister(AlertsCreatedTotal)
	prometheus.MustRegister(AlertsResolvedTotal)
}
