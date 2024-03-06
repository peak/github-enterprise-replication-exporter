package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"

	"github.com/peak/picolo"
	"github.com/prometheus/client_golang/prometheus"
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
	scrapeError = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "status_scrape_error_total",
		Help:      "Total number of error while scraping.",
	})
	version string = "development"
)

type Exporter struct {
	replStatus *string
	role       string
	status     []byte
	locker     uint32
	logger     *picolo.Logger
}

// NewExporter returns an initialized Exporter.
func NewExporter(binaryPath *string, logLevel *string) *Exporter {

	picoloLogLevel, err := picolo.LevelFromString(*logLevel)
	if err != nil {
		picoloLogLevel, _ = picolo.LevelFromString("info")
	}

	return &Exporter{
		replStatus: binaryPath,
		logger: picolo.New(
			picolo.WithPrefix("github-enterprise-replication-exporter:"),
			picolo.WithLevel(picoloLogLevel),
		),
	}
}

func (e *Exporter) checkReplication() error {
	if !atomic.CompareAndSwapUint32(&e.locker, 0, 1) {
		return nil
	}
	defer atomic.StoreUint32(&e.locker, 0)
	if e.role == "replica" {
		status, err := exec.Command(*e.replStatus).Output()
		if err != nil {
			return fmt.Errorf("error during replication check while running %v: %s", *e.replStatus, err)
		}
		e.status = status
	}
	return nil
}

func (e *Exporter) setRole() {
	if _, err := os.Stat(*e.replStatus); os.IsNotExist(err) {
		e.logger.Errorf("ghe-repl-status not found on path: %v", *e.replStatus)
		os.Exit(1)
	}

	cmdArgs := []string{"-r"}
	role, err := exec.Command(*e.replStatus, cmdArgs...).Output()
	if err != nil {
		e.logger.Errorf("Error running %v: %s", *e.replStatus, err)
		os.Exit(1)
	}
	e.role = strings.TrimSuffix(string(role), "\n")
}

// Describe describes all the metrics ever exported by the GHE Replication exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- service
	ch <- scrapeError.Desc()
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1, e.role,
	)

	if err := e.checkReplication(); err != nil {
		scrapeError.Inc()
		ch <- scrapeError
		e.logger.Errorf("Scrape error: %s", err)
	}

	for _, line := range strings.Split(string(e.status), "\n") {
		l := strings.Split(line, " ")
		if len(l) < 2 {
			// We hit empty line, just skip
			continue
		} 
		if l[0] == "Verifying" {
			// This is a new line, which can be skiped
			continue
		}
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

func main() {
	var (
		listenAddress     = flag.String("listen-address", ":9169", "Address to listen on for web interface and telemetry")
		metricsPath       = flag.String("metrics-path", "/metrics", "Path under which to expose metrics")
		gheReplStatusPath = flag.String("ghe-repl-status-path", "/usr/local/bin/ghe-repl-status", "Path where ghe-repl-status can be found")
		logLevel          = flag.String("log-level", "info", "Log level (debug/info/warning/error)")
		checkVersion      = flag.Bool("version", false, "Prints version")
	)

	flag.Parse()

	if *checkVersion {
		fmt.Printf("Version: %q", version)
		return
	}

	exporter := NewExporter(gheReplStatusPath, logLevel)
	exporter.setRole()
	exporter.logger.Infof("Starting github_replication_exporter, version: %s", version)

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
             </body>
             </html>`))
	})

	exporter.logger.Infof("Listening on %s", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, nil); err != nil {
		exporter.logger.Errorf("%s", err)
		os.Exit(1)
	}
}
