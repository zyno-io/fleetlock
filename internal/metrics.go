package fleetlock

import (
	"github.com/prometheus/client_golang/prometheus"
)

// fleetlock Prometheus metrics
type metrics struct {
	lockState        *prometheus.GaugeVec
	lockTransitions  *prometheus.GaugeVec
	lockRequests     prometheus.Counter
	unlockRequests   prometheus.Counter
	cooldownDenials  *prometheus.CounterVec
	rebootWithinLock *prometheus.CounterVec
}

// newMetrics creates fleetlock Prometheus metrics.
func newMetrics() *metrics {
	lockState := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fleetlock_lock_state",
		Help: "State of the fleetlock lease (0 unlocked, 1 locked)",
	}, []string{"group"})

	lockTransitions := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fleetlock_lock_transition_count",
		Help: "Number of fleetlock lease transitions",
	}, []string{"group"})

	lockRequests := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "fleetlock_lock_request_count",
		Help: "Number of lock requests",
	})

	unlockRequests := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "fleetlock_unlock_request_count",
		Help: "Number of unlock requests",
	})

	cooldownDenials := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleetlock_cooldown_denial_count",
		Help: "Number of lock requests denied due to the post-reboot cooldown",
	}, []string{"group"})

	rebootWithinLock := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fleetlock_reboot_within_lock_count",
		Help: "Number of unlocks where the node rebooted during its lock window",
	}, []string{"group"})

	return &metrics{
		lockState:        lockState,
		lockTransitions:  lockTransitions,
		lockRequests:     lockRequests,
		unlockRequests:   unlockRequests,
		cooldownDenials:  cooldownDenials,
		rebootWithinLock: rebootWithinLock,
	}
}

// Register registers metrics on the given registry.
func (m *metrics) Register(registry prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.lockState,
		m.lockTransitions,
		m.lockRequests,
		m.unlockRequests,
		m.cooldownDenials,
		m.rebootWithinLock,
	}

	return registerAll(registry, collectors...)
}

// registerAll registers all Prometheus collectors on the Prometheus Registerer
// or returns an error.
func registerAll(registry prometheus.Registerer, collectors ...prometheus.Collector) error {
	for _, collector := range collectors {
		if err := registry.Register(collector); err != nil {
			return err
		}
	}
	return nil
}
