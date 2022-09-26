package main

import (
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
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
		log.Fatalf("ghe-repl-status not found on path: %v", *binaryPath)
	}
	// Maybe implement other black magic for checks
	cmdArgs := []string{"-r"}
	role, err := exec.Command(*binaryPath, cmdArgs...).Output()
	if err != nil {
		log.Fatalf("Error running %v: %s", *binaryPath, err)
	}
	log.Debugf("The role of GHE server is %s", role)
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
		log.Fatalf("Error during replication check while running %v: %s", *e.replStatus, err)
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
	log.Debugf("Status output: %s", string(e.status))

	for _, line := range strings.Split(string(e.status), "\n") {
		l := strings.Split(line, " ")
		if len(l) < 2 {
			// We hit empty line, just skip
			continue
		}
		log.Debugf("Parsed: %s", l)
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
		listenAddress     = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9169").String()
		metricsPath       = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		gheReplStatusPath = kingpin.Flag("ghe.ReplStatusPath", "Path where ghe-repl-status can be found.").Default("/usr/local/bin/ghe-repl-status").String()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("github_replication_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infof("Starting github_replication_exporter, ", version.Info())
	log.Infof("Build context: ", version.BuildContext())

	exporter, err := NewExporter(gheReplStatusPath)
	if err != nil {
		log.Fatalln(err)
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

	log.Infof("Listening on %s", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
