#  Github Enterprise Replication Exporter

This application is designed to parse ghe-repl-status and export it as Prometheus metrics

## Build

- Binary build:

```shell
go build -ldflags="-X 'main.version=`git describe --tags --abbrev=0`'" .
```

- Docker build:

```shell
docker build -t peakcom/github-enterprise-replication-exporter .
```

## Usage

```
Usage of /github-enterprise-replication-exporter:
  -ghe-repl-status-path string
        Path where ghe-repl-status can be found (default "/usr/local/bin/ghe-repl-status")
  -listen-address string
        Address to listen on for web interface and telemetry (default ":9169")
  -log-level string
        Log level (debug/info/warning/error) (default "info")
  -metrics-path string
        Path under which to expose metrics (default "/metrics")
  -version
        Prints version
```