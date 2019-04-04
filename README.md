#  Github Enterprise Replication Exporter

This application is designed to parse ghe-repl-status and export it as prometheus metrics. It is our current way of scraping status.

#### Exported information

Exports 2 metrics

###### github_replication_exporter_replicated_service

All replicated services are parsed and returned. 0 means that service is healthy, and 1 is unhealthy.

| label   | value  |  
|---|---|
| role  | primary or replica | 
| service | any replicated service returned by ghe-repl-status |


###### github_replication_exporter_up 

The gole of this metric would be to monitor that replica role is 0. Primary node throws an error, and 1 is given back, but it is still possible to see that primary node has also exporter running.

| label   | value  |  
|---|---|
| role  | primary or replica | 

####  Prometheus Alert examples

```bazaar
TBD
```

#### Docker

You can test quickly with running following command:

`docker run peakcom/github-enterprise-replication-exporter:latest`