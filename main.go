package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/prometheus/prompb"

	dto "github.com/prometheus/client_model/go"
)

func main() {
	url := flag.String("url", "", "The prometheus remote write url")
	tlsCAFile := flag.String("tls-ca-file", "", "The prometheus remote write TLS ca file")
	tlsKeyFile := flag.String("tls-key-file", "", "The prometheus remote write TLS key file")
	tlsCertFile := flag.String("tls-cert-file", "", "The prometheus remote write TLS cert file")
	tlsSkipVerify := flag.Bool("tls-skip-verify", false, "Disables the prometheus remote write TLS verify")
	timeout := flag.Int("timeout", 30, "The prometheus remote write timeout")
	debug := flag.Bool("debug", false, "Enable verbose mode")
	flag.Parse()

	if len(*url) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	mf, err := parseReader(bufio.NewReader(os.Stdin))
	if err != nil {
		log.Fatal(err)
	}
	wr := formatData(mf)

	client := initHTTPClient(*tlsCAFile, *tlsKeyFile, *tlsCertFile, *tlsSkipVerify, *timeout)
	sendData(client, wr, *url, *debug)
}

// formatData convert metric family to a writerequest
func formatData(mf map[string]*dto.MetricFamily) *prompb.WriteRequest {
	wr := &prompb.WriteRequest{
		Timeseries: []*prompb.TimeSeries{},
	}

	for metricName, data := range mf {
		for _, metric := range data.Metric {

			// add the metric name
			timeserie := &prompb.TimeSeries{
				Labels: []*prompb.Label{
					&prompb.Label{
						Name:  "__name__",
						Value: metricName,
					},
				},
			}

			for _, label := range metric.Label {
				timeserie.Labels = append(timeserie.Labels, &prompb.Label{
					Name:  label.GetName(),
					Value: label.GetValue(),
				})
			}

			timeserie.Samples = []prompb.Sample{
				prompb.Sample{
					Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
					Value:     getValue(metric),
				},
			}

			wr.Timeseries = append(wr.Timeseries, timeserie)
		}
	}
	return wr
}

// sendData push timeseries to a remote write url
func sendData(client *http.Client, wr *prompb.WriteRequest, url string, debug bool) {
	if debug {
		log.Printf("Sending %d timeseries", len(wr.Timeseries))
	}

	// Marshal the data into a byte slice using the protobuf library.
	data, err := proto.Marshal(wr)
	if err != nil {
		log.Fatal(err)
	}

	// Encode the content into snappy encoding.
	compressed := snappy.Encode(nil, data)

	// Create an HTTP request from the body content and set necessary parameters.
	req, err := http.NewRequest("POST", url, bytes.NewReader(compressed))
	if err != nil {
		log.Fatal(err)
	}

	// Set the required HTTP header content.
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	if debug {
		requestDump, err := httputil.DumpRequest(req, true)
		if err != nil {
			log.Fatal(err)
		}
		log.Println(string(requestDump))
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	log.Printf(
		"method=POST url=%s length=%d status=%d duration=%d",
		url,
		req.ContentLength,
		resp.StatusCode,
		int(time.Since(start).Milliseconds()),
	)

	if resp.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		if debug {
			responseDump, err := httputil.DumpResponse(resp, true)
			if err != nil {
				log.Fatal(err)
			}
			log.Println(string(responseDump))
		}
		log.Fatalf("Unable to push timeseries: %d - %s", resp.StatusCode, string(bodyBytes))
	}
}

// ParseReader consumes an io.Reader and returns the MetricFamily
func parseReader(input io.Reader) (map[string]*dto.MetricFamily, error) {
	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(input)
	if err != nil {
		return nil, err
	}
	return mf, nil
}

// getValue return the value of a timeserie without the need to give value type
func getValue(m *dto.Metric) float64 {
	switch {
	case m.Gauge != nil:
		return m.GetGauge().GetValue()
	case m.Counter != nil:
		return m.GetCounter().GetValue()
	case m.Untyped != nil:
		return m.GetUntyped().GetValue()
	default:
		return 0.
	}
}

func initHTTPClient(caFile string, certFile, keyFile string, insecure bool, timeout int) *http.Client {
	tlsConfig := &tls.Config{}

	if insecure {
		tlsConfig.InsecureSkipVerify = insecure
	}

	caCertPool := x509.NewCertPool()
	if caFile != "" {
		caCert, err := ioutil.ReadFile(caFile)
		if err != nil {
			log.Fatal(err)
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}

	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Fatal(err)
		}

		tlsConfig.RootCAs = caCertPool
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
}
