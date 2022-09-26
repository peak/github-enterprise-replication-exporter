#  Github Enterprise Replication Exporter

This application is designed to parse ghe-repl-status and export it as Prometheus metrics

## Build

- Binary build:

```shell
go build .
```

- Docker build:

```shell
docker build -t peakcom/github-enterprise-replication-exporter .
```

## Usage

```
usage: github-enterprise-replication-exporter [<flags>]

Flags:
  -h, --help              Show context-sensitive help (also try --help-long and --help-man).
      --web.listen-address=":9169"  
                          Address to listen on for web interface and telemetry.
      --web.telemetry-path="/metrics"  
                          Path under which to expose metrics.
      --ghe.ReplStatusPath="/usr/local/bin/ghe-repl-status"  
                          Path where ghe-repl-status can be found.
      --log.level="info"  Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]
      --log.format="logger:stderr"  
                          Set the log target and format. Example: "logger:syslog?appname=bob&local=7" or "logger:stdout?json=true"
      --version           Show application version.
```