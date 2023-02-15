// A cli to send Prometheus timeseries to a remote write.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/fgouteroux/prom-push-cli/pkg/client"
	"github.com/fgouteroux/prom-push-cli/pkg/metrics"
)

var (
	CliVersion = "0.1.4"
)

// Headers is string slice
type Headers []string

func (ml *Headers) String() string {
	return fmt.Sprintln(*ml)
}

// Set string value in Headers
func (ml *Headers) Set(s string) error {
	*ml = append(*ml, s)
	return nil
}

func main() {
	var headers Headers
	flag.Var(&headers, "header", "The prometheus remote write header like key=\"value\", repeatable")

	url := flag.String("url", "", "The prometheus remote write url")
	tlsCAFile := flag.String("tls-ca-file", "", "The prometheus remote write TLS ca file")
	tlsKeyFile := flag.String("tls-key-file", "", "The prometheus remote write TLS key file")
	tlsCertFile := flag.String("tls-cert-file", "", "The prometheus remote write TLS cert file")
	tlsSkipVerify := flag.Bool("tls-skip-verify", false, "Disables the prometheus remote write TLS verify")
	jobLabel := flag.String("job-label", "prom-push-cli", "The prometheus remote write job label to add")
	timeout := flag.Int("timeout", 30, "The prometheus remote write timeout")
	debug := flag.Bool("debug", false, "Enable verbose mode")
	enableHTTP2 := flag.Bool("enable-http2", false, "Enables http2")
	version := flag.Bool("version", false, " Show version")
	flag.Parse()

	if *version {
		fmt.Println(CliVersion)
		os.Exit(0)
	}

	if len(*url) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	data, err := metrics.ParseAndFormat(bufio.NewReader(os.Stdin), *jobLabel)
	if err != nil {
		log.Fatal(err)
	}

	var promClient *client.Client
	if *tlsCAFile == "" && *tlsKeyFile == "" && *tlsCertFile == "" && *tlsSkipVerify == false {
		promClient = client.Configure(*url, *debug, *timeout, headers)
	} else {
		promClient = client.ConfigureWithTLS(*url, *tlsCAFile, *tlsKeyFile, *tlsCertFile, *enableHTTP2, *debug, *tlsSkipVerify, *timeout, headers)
	}
	err = promClient.PushWithRetries(data)
	if err != nil {
		log.Fatal(err)
	}
}
