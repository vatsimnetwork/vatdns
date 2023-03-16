package dnshaiku

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

type RateCollector struct {
	DnsRate *prometheus.Desc
}

func newRateCollector() *RateCollector {
	return &RateCollector{
		DnsRate: prometheus.NewDesc("vatdns_dnshaiku_requests_per_second",
			"RPS DNS server is currently processing.",
			nil, nil,
		),
	}
}

func (collector *RateCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.DnsRate
	//ch <- collector.HttpRate
}

func (collector *RateCollector) Collect(ch chan<- prometheus.Metric) {
	m1 := prometheus.MustNewConstMetric(collector.DnsRate, prometheus.CounterValue, float64(dnsRateCounter.Rate()))
	m1 = prometheus.NewMetricWithTimestamp(time.Now(), m1)
	ch <- m1
}
