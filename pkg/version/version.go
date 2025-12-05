package version

import (
	"fmt"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"

	info = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gluetarr_info",
			Help: "Information about the Gluetarr build",
		},
		[]string{"version", "commit", "date", "go_version"},
	)
)

func init() {
	info.WithLabelValues(Version, Commit, Date, runtime.Version()).Set(1)
}

func String() string {
	return fmt.Sprintf("Gluetarr %s (commit: %s, built: %s)", Version, Commit, Date)
}
