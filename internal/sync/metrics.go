package sync

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	currentPort = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "forwardarr_current_port",
		Help: "The current forwarded port being synchronized",
	})

	syncTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "forwardarr_sync_total",
		Help: "Total number of successful port sync operations",
	})

	syncErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "forwardarr_sync_errors",
		Help: "Total number of failed port sync operations",
	})

	lastSyncTimestamp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "forwardarr_last_sync_timestamp",
		Help: "Unix timestamp of the last successful sync",
	})
)

func SetCurrentPort(port int) {
	currentPort.Set(float64(port))
}

func IncrementSyncTotal() {
	syncTotal.Inc()
}

func IncrementSyncErrors() {
	syncErrors.Inc()
}

func UpdateLastSyncTimestamp() {
	lastSyncTimestamp.Set(float64(time.Now().Unix()))
}
