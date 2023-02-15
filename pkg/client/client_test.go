// A cli to send Prometheus timeseries to a remote write.
package client_test

import (
	"fmt"
	"github.com/fgouteroux/prom-push-cli/pkg/client"
	"github.com/fgouteroux/prom-push-cli/pkg/metrics"
	"log"
)

func ExampleConfigure() {
	// Configure the client
	promClient := client.Configure(
		"http://my-remote-write:10001/api/v1/push",
		false,
		10,
		make([]string),
	)
}

func ExampleConfigureWithInsecureTLS() {
	// Configure the client
	promClient := client.ConfigureWithInsecureTLS(
		"https://my-remote-write:10001/api/v1/push",
		false,
		false,
		10,
		make([]string),
	)
}

func ExampleConfigureWithTLS() {
	// Configure the client with TLS setting
	promClient := client.Configure(
		"http://my-remote-write:10001/api/v1/push",
		"/tmp/caFile",
		"/tmp/keyFile",
		"/tmp/certFile",
		false,
		false,
		false,
		10,
		make([]string),
	)
}

func ExamplePushWithRetries() {
	// Push timeseries with a backoff retry in case of errors
	err = promClient.PushWithRetries(&prompb.WriteRequest{})
	if err != nil {
		log.Fatal(err)
	}
}

func ExamplePush() {
	// Push timeseries with a backoff retry in case of errors
	err := promClient.Push(&prompb.WriteRequest{})
	if err != nil {
		log.Fatal(err)
	}
}
