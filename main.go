package main

import (
	"flag"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"

	"github.com/peak/picolo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
)

const (
	namespace = "github_replication_exporter"
)

var (
	up = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "up"),
		"Was the last query of Github Replication successful.",
		[]string{"role"}, nil,
	)
	service = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "replicated_service"),
		"Replicated service status",
		[]string{"service", "role"}, nil,
	)
	logger *picolo.Logger
)

type Exporter struct {
	replStatus *string
	role       string
	status     []byte
	locker     uint32
}

// NewExporter returns an initialized Exporter.
func NewExporter(binaryPath *string) (*Exporter, error) {
	if _, err := os.Stat(*binaryPath); os.IsNotExist(err) {
		logger.Errorf("ghe-repl-status not found on path: %v", *binaryPath)
	}
	// Maybe implement other black magic for checks
	cmdArgs := []string{"-r"}
	role, err := exec.Command(*binaryPath, cmdArgs...).Output()
	if err != nil {
		logger.Errorf("Error running %v: %s", *binaryPath, err)
	}
	logger.Debugf("The role of GHE server is %s", role)
	return &Exporter{
		replStatus: binaryPath,
		role:       strings.TrimSuffix(string(role), "\n"),
	}, nil
}

func (e *Exporter) checkReplication() {
	if !atomic.CompareAndSwapUint32(&e.locker, 0, 1) {
		return
	}
	defer atomic.StoreUint32(&e.locker, 0)
	status, err := exec.Command(*e.replStatus).Output()
	if err != nil {
		logger.Errorf("Error during replication check while running %v: %s", *e.replStatus, err)
	}
	e.status = status
}

// Describe describes all the metrics ever exported by the GHE Replication exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- service
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1, e.role,
	)

	e.checkReplication()
	logger.Debugf("Status output: %s", string(e.status))

	for _, line := range strings.Split(string(e.status), "\n") {
		l := strings.Split(line, " ")
		if len(l) < 2 {
			// We hit empty line, just skip
			continue
		}
		logger.Debugf("Parsed: %s", l)
		if l[0] == "OK:" {
			ch <- prometheus.MustNewConstMetric(
				service, prometheus.GaugeValue, 1, l[1], e.role,
			)
		} else {
			ch <- prometheus.MustNewConstMetric(
				service, prometheus.GaugeValue, 0, l[1], e.role,
			)
		}
	}
}

func init() {
	prometheus.MustRegister(version.NewCollector("github_replication_exporter"))
}

func main() {
	var (
		listenAddress     = flag.String("listen-address", ":9169", "Address to listen on for web interface and telemetry")
		metricsPath       = flag.String("metrics-path", "/metrics", "Path under which to expose metrics")
		gheReplStatusPath = flag.String("ghe-repl-status-path", "/usr/local/bin/ghe-repl-status", "Path where ghe-repl-status can be found")
		logLevel          = flag.String("log-level", "info", "Log level (debug/info/warning/error)")
	)

	flag.Parse()

	picoloLogLevel, _ := picolo.LevelFromString(*logLevel)

	logger := picolo.New(
		picolo.WithPrefix("github-enterprise-replication-exporter:"),
		picolo.WithLevel(picoloLogLevel),
	)

	logger.Infof("Starting github_replication_exporter, version: ", version.Info())

	exporter, err := NewExporter(gheReplStatusPath)
	if err != nil {
		logger.Errorf("Error creating new exporter", err)
	}
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Github Replication Exporter</title></head>
             <body>
             <h1>Github Replication Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             <h2>Options</h2>
             </dl>
             <h2>Build</h2>
             <pre>` + version.Info() + ` ` + version.BuildContext() + `</pre>
             </body>
             </html>`))
	})

	logger.Infof("Listening on %s", *listenAddress)
	logger.Errorf("%s", http.ListenAndServe(*listenAddress, nil))
}
