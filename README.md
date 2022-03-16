# Prometheus Push Cli

The prom-push-cli is made for testing purpose and for jobs which expose
their metrics to Prometheus without scraping method like stdout or a custom file.

Note: the node exporter text-file collector could be used if metrics are related to the node.

The prom-push-cli is not an alternative to:
- [prometheus agent](https://prometheus.io/blog/2021/11/16/agent/)
- [grafana-agent](https://github.com/grafana/agent)
- [PushGateway](https://github.com/prometheus/pushgateway)


## How it works

The prom-push-cli read input from stdin and use [expfmt](https://pkg.go.dev/github.com/prometheus/common@v0.15.0/expfmt#TextParser.TextToMetricFamilies) to extract the timeseries from flat text-based exchange format and creates MetricFamily proto messages.

Then it create a WriteRequest object, serialize it and send as an HTTP request with snappy-compressed protocol buffer.

This help to be a bit more performant with reducing the payload size.

## Usage

```shell
Usage of prom-push-cli:
  -debug
    	Enable verbose mode
  -timeout int
    	The prometheus remote write timeout (default 30)
  -tls-ca-file string
    	The prometheus remote write TLS ca file
  -tls-cert-file string
    	The prometheus remote write TLS cert file
  -tls-key-file string
    	The prometheus remote write TLS key file
  -tls-skip-verify
    	Disables the prometheus remote write TLS verify
  -url string
    	The prometheus remote write url
```

## Run it

### Without TLS

```shell
echo 'custom_metric_info{job="manual"} 1.' | ./prom-push-cli -url http://my-remote-write:10001/api/v1/receive
```

### With TLS Insecure
```shell
echo 'custom_metric_info{job="manual"} 1.' | ./prom-push-cli -url https://my-remote-write:10001/api/v1/receive -tls-skip-verify
```

### With TLS
```shell
echo 'custom_metric_info{job="manual"} 1.' | ./prom-push-cli -url http://my-remote-write:10001/api/v1/receive -tls-cert-file mycert.crt -tls-key-file my-key.pem -tls-ca-file myca.crt
```

### With Debug mode
```shell
cat /tmp/metric 
custom_metric_info{job="manual"} 1
custom_metric_info{job="manual2"} 1


cat /tmp/metric | ./prom-push-cli -url http://my-remote-write:10001/api/v1/receive -debug
2022/03/16 11:48:48 Sending 2 timeseries
2022/03/16 11:48:48 POST /api/v1/receive HTTP/1.1
Host: my-remote-write:10001
Content-Encoding: snappy
Content-Type: application/x-protobuf
X-Prometheus-Remote-Write-Version: 0.1.0

��F
A

__name__custom_metric_info

jobmanual	�?�¼��/
B
zC
C	C2FD
2022/03/16 11:48:48 method=POST url=http://my-remote-write:10001/api/v1/receive length=92 status=200 duration=40
```

## TODO

- [ ] add tests...


## License

Licensed under the terms of the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0).

## References

Inspired by:
- https://github.com/timescale/promscale/blob/master/docs/writing_to_promscale.md
- https://stackoverflow.com/questions/65388098/how-to-parse-prometheus-data
