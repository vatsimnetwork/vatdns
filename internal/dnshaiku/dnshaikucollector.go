package dnshaiku

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

type RateCollector struct {
	Rate *prometheus.Desc
}

func newRateCollector() *RateCollector {
	return &RateCollector{
		Rate: prometheus.NewDesc("vatdns_dnshaiku_requests_per_second",
			"RPS DNS server is currently processing.",
			nil, nil,
		),
	}
}

func (collector *RateCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.Rate
}

func (collector *RateCollector) Collect(ch chan<- prometheus.Metric) {
	m1 := prometheus.MustNewConstMetric(collector.Rate, prometheus.CounterValue, float64(dnsRateCounter.Rate()))
	m1 = prometheus.NewMetricWithTimestamp(time.Now(), m1)
	ch <- m1
}
