// Package Client
package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"golang.org/x/net/http2"
)

// A backoff schedule for when and how often to retry failed HTTP requests
var backoffSchedule = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

// Client config
type Client struct {
	HTTP    *http.Client
	Debug   bool
	Headers []string
	URL     string
}

// Configure client
func Configure(url string, debug bool, timeout int, headers []string) *Client {
	return &Client{
		HTTP:    &http.Client{Timeout: time.Duration(timeout) * time.Second},
		Debug:   debug,
		Headers: headers,
		URL:     url,
	}
}

// Configure client with SSL
func ConfigureWithTLS(url string, caFile string, keyFile, certFile string, enableHTTP2, insecure bool, debug bool, timeout int, headers []string) *Client {
	return &Client{
		HTTP:    initHTTPClient(url, caFile, keyFile, certFile, enableHTTP2, insecure, timeout),
		Debug:   debug,
		Headers: headers,
		URL:     url,
	}
}

// Configure client with insecure SSL
func ConfigureWithInsecureTLS(url string, enableHTTP2, debug bool, timeout int, headers []string) *Client {
	return &Client{
		HTTP:    initHTTPClient(url, "", "", "", enableHTTP2, true, timeout),
		Debug:   debug,
		Headers: headers,
		URL:     url,
	}
}

// Initialize HTTP Client
func initHTTPClient(url, caFile string, keyFile, certFile string, enableHTTP2, insecure bool, timeout int) *http.Client {
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

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	if strings.HasPrefix(url, "https") && enableHTTP2 {
		client.Transport = &http2.Transport{TLSClientConfig: tlsConfig}
	} else {
		client.Transport = &http.Transport{TLSClientConfig: tlsConfig}
	}

	return client
}

// Push timeseries to a remote write url
func (client Client) Push(wr *prompb.WriteRequest) error {

	if client.Debug {
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
	req, err := http.NewRequest("POST", client.URL, bytes.NewReader(compressed))
	if err != nil {
		log.Fatal(err)
	}

	// Set custom HTTP headers
	for _, h := range client.Headers {
		header := strings.Split(h, "=")
		req.Header[header[0]] = []string{header[1]}
	}

	// Set the required HTTP header content.
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	if client.Debug {
		requestDump, err := httputil.DumpRequest(req, false)
		if err != nil {
			return err
		}
		log.Printf("request: \n%s", string(requestDump))
		log.Printf("request body: \n%s", proto.MarshalTextString(wr))
	}

	start := time.Now()
	resp, err := client.HTTP.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	log.Printf(
		"method=POST url=%s length=%d status=%d duration=%d",
		client.URL,
		req.ContentLength,
		resp.StatusCode,
		int(time.Since(start).Milliseconds()),
	)

	if client.Debug {
		responseDump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
		log.Printf("response: \n%s", string(responseDump))
	}

	if resp.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Unable to push timeseries: %d - %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

// Push with retries timeseries to a remote write url
func (client Client) PushWithRetries(wr *prompb.WriteRequest) error {
	var err error
	for _, backoff := range backoffSchedule {
		err = client.Push(wr)
		if err == nil {
			break
		}

		log.Printf("Request error - %+v\n", err)
		log.Printf("Retrying in %v\n", backoff)
		time.Sleep(backoff)
	}

	return err
}
