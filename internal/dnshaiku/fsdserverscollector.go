package dnshaiku

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vatsimnetwork/vatdns/pkg/common"
	"time"
)

type FsdServersCollector struct {
	CurrentUsers         *prometheus.Desc
	MaxUsers             *prometheus.Desc
	AcceptingConnections *prometheus.Desc
	RemainingSlots       *prometheus.Desc
	Name                 string
}

func newFsdServersCollector(fsdServer *common.FSDServer) *FsdServersCollector {
	return &FsdServersCollector{
		Name: fsdServer.Name,
		CurrentUsers: prometheus.NewDesc("vatdns_dnshaiku_current_users",
			"Current amount of users connected server.",
			nil, prometheus.Labels{"server": fsdServer.Name},
		),
		MaxUsers: prometheus.NewDesc("vatdns_dnshaiku_max_users",
			"Maximum amount of users a server will allow at a time.",
			nil, prometheus.Labels{"server": fsdServer.Name},
		),
		AcceptingConnections: prometheus.NewDesc("vatdns_dnshaiku_accepting_connections",
			"If server is in rotation to be considered for connections",
			nil, prometheus.Labels{"server": fsdServer.Name},
		),
		RemainingSlots: prometheus.NewDesc("vatdns_dnshaiku_remaining_slots",
			"Remaining slots on a server",
			nil, prometheus.Labels{"server": fsdServer.Name},
		),
	}
}

func (collector FsdServersCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.CurrentUsers
	ch <- collector.MaxUsers
	ch <- collector.AcceptingConnections
	ch <- collector.RemainingSlots
}

func (collector FsdServersCollector) Collect(ch chan<- prometheus.Metric) {

	//Note that you can pass CounterValue, GaugeValue, or UntypedValue types here.
	fsdServer, _ := fsdServers.Load(collector.Name)
	fsdServerStruct := fsdServer.(*common.FSDServer)
	m1 := prometheus.MustNewConstMetric(collector.CurrentUsers, prometheus.CounterValue, float64(fsdServerStruct.CurrentUsers))
	m2 := prometheus.MustNewConstMetric(collector.MaxUsers, prometheus.CounterValue, float64(fsdServerStruct.MaxUsers))
	m3 := prometheus.MustNewConstMetric(collector.AcceptingConnections, prometheus.CounterValue, float64(fsdServerStruct.AcceptingConnections()))
	m4 := prometheus.MustNewConstMetric(collector.RemainingSlots, prometheus.CounterValue, float64(fsdServerStruct.RemainingSlots))
	m1 = prometheus.NewMetricWithTimestamp(time.Now(), m1)
	m2 = prometheus.NewMetricWithTimestamp(time.Now(), m2)
	m3 = prometheus.NewMetricWithTimestamp(time.Now(), m3)
	m4 = prometheus.NewMetricWithTimestamp(time.Now(), m4)
	ch <- m1
	ch <- m2
	ch <- m3
	ch <- m4
}
