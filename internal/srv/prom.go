package srv

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystem = "load_balancer_operator"

var (
	numberLoadBalancersCreatedGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: subsystem,
			Name:      "load_balancers_created",
			Help:      "Total count of load balancers created",
		},
	)
	numberLoadBalancersDeletedGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: subsystem,
			Name:      "load_balancers_deleted",
			Help:      "Total count of load balancers deleted",
		},
	)
)
