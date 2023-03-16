package dnshaiku

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/oschwald/geoip2-golang"
	"github.com/paulbellamy/ratecounter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"log"
	"net"
	"net/http"
	"sync"
	"vatdns/internal/logger"
)

var (
	fsdServers     sync.Map
	db             *geoip2.Reader
	dnsRateCounter *ratecounter.RateCounter
	dnsIpOverride  string
	publicIp       string
)

func Main() {
	// Get public IP for machine from interfaces...hope this works. Works on Droplets.
	// We need this for when someone queries what IPs to make an HTTP request to for
	// fsd-http.connect.vatsim.net
	addrs, _ := net.InterfaceAddrs()
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsPrivate() {
			if ipnet.IP.To4() != nil {
				publicIp = ipnet.IP.String()
			}
		}
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              viper.GetString("SENTRY_DSN"),
		TracesSampleRate: 0,
	})
	if err != nil {
		logger.Info(fmt.Sprintf("sentry.Init: %s", err))
	}
	// Handle various web things
	go StartDataWebServer()
	// Starts dataprocessor and waits for data before starting
	go dataProcessorManager()
	// Starts a tcp+udp DNS server
	go StartDnsServer()
	// Starts an HTTP server to return an IP to connect to
	go StartWebServer()
	// Starts a Prometheus exporter endpoint to be scraped
	go http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", viper.GetString("PROMETHEUS_METRICS_PORT")), nil))

}
