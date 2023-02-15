// Package Metrics
package metrics

import (
	"io"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/prometheus/prompb"

	dto "github.com/prometheus/client_model/go"
)

// formatData convert metric family to a writerequest
func formatData(mf map[string]*dto.MetricFamily, jobLabel string) *prompb.WriteRequest {
	wr := &prompb.WriteRequest{}

	for metricName, data := range mf {
		for _, metric := range data.Metric {

			// add the metric name
			timeserie := prompb.TimeSeries{
				Labels: []prompb.Label{
					prompb.Label{
						Name:  "__name__",
						Value: metricName,
					},
					prompb.Label{
						Name:  "job",
						Value: jobLabel,
					},
				},
			}

			for _, label := range metric.Label {
				labelname := label.GetName()
				if labelname == "job" {
					continue
				}
				timeserie.Labels = append(timeserie.Labels, prompb.Label{
					Name:  labelname,
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

// parseReader consumes an io.Reader and returns the MetricFamily
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

// ParseAndFormat return the data in the expected prometheus metrics write request format
func ParseAndFormat(input io.Reader, jobLabel string) (*prompb.WriteRequest, error) {
	mf, err := parseReader(input)
	if err != nil {
		return nil, err
	}
	return formatData(mf, jobLabel), nil
}
