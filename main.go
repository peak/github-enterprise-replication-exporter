package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
}

// NewExporter returns an initialized Exporter.
func NewExporter(binaryPath *string) (*Exporter, error) {
	log.Debugln("Checking env")
	if _, err := os.Stat(*binaryPath); os.IsNotExist(err) {
		log.Fatalln("ghe-repl-status cound not be found")
	}
	// Maybe implement other black magic for checks
	cmdArgs := []string{"-r"}
	role, err := exec.Command(*binaryPath, cmdArgs...).Output()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("The role of GHE server is %s", role)
	return &Exporter{
		replStatus: binaryPath,
		role:       strings.TrimSuffix(string(role), "\n"),
	}, nil
}

// Describe describes all the metrics ever exported by the Aviatrix exporter. It
// implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- up
	ch <- service
}

// Collect fetches the stats from configured Aviatrix location and delivers them
// as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	status, err := exec.Command(*e.replStatus).Output()
	var retValue float64
	retValue = 0
	if err != nil {
		retValue = 1
	}
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, retValue, e.role,
	)
	log.Debugln(string(status))
	parsed := strings.Split(string(status), "\n")
	for _, line := range parsed {
		l := strings.Split(line, " ")
		if len(l) < 2 {
			log.Debugln("We hit empty line, just skip")
			continue
		}
		log.Debugln(l)
		var serviceRetValue float64
		if l[0] == "OK:" {
			serviceRetValue = 0;
		} else {
			serviceRetValue = 1;
		}
		ch <- prometheus.MustNewConstMetric(
			service, prometheus.GaugeValue, serviceRetValue, l[1], e.role,
		)
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

	log.Infoln("Starting github_replication_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

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

	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
