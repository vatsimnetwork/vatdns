package dnshaiku

import (
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/jftuga/geodist"
	"github.com/miekg/dns"
	"github.com/oschwald/geoip2-golang"
	"github.com/paulbellamy/ratecounter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"io"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
	"vatdns/internal/logger"
	"vatdns/pkg/common"
)

var (
	fsdServers     sync.Map
	db             *geoip2.Reader
	dnsRateCounter *ratecounter.RateCounter
	dnsIpOverride  string
)

func Main() {
	// Handle various web things
	go handleWebRequests()
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              viper.GetString("SENTRY_DSN"),
		TracesSampleRate: 0,
	})
	if err != nil {
		logger.Info(fmt.Sprintf("sentry.Init: %s", err))
	}
	// Starts dataprocessor and waits for data before starting
	go dataProcessorManager()
	// Starts a tcp+udp DNS server
	go startDnsServer()
	// Starts a Prometheus exporter endpoint to be scraped
	go http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", viper.GetString("PROMETHEUS_METRICS_PORT")), nil))

}

func pickServerToReturn(sourceIpLatLng geodist.Coord) *common.FSDServer {
	// Slices are easier for sorting
	initialServers := make([]common.FSDServer, 0)
	finalServers := make([]common.FSDServer, 0)

	// Get servers into a slice, skipping those that are not accepting connections
	fsdServers.Range(func(k, v interface{}) bool {
		fsdServerStruct := v.(*common.FSDServer)
		if fsdServerStruct.AcceptingConnections() == 0 {
			return true
		}
		miles, _, _ := geodist.VincentyDistance(sourceIpLatLng, geodist.Coord{Lat: fsdServerStruct.Latitude, Lon: fsdServerStruct.Longitude})
		initialServers = append(initialServers, common.FSDServer{
			Name:           fsdServerStruct.Name,
			Distance:       miles,
			RemainingSlots: fsdServerStruct.RemainingSlots,
			IpAddress:      fsdServerStruct.IpAddress,
			Country:        fsdServerStruct.Country,
		})
		return true
	})

	if len(initialServers) == 0 {
		logger.Error("No servers possible for a request, using default FSD server")
		fsdServer, _ := fsdServers.Load(viper.GetString("DEFAULT_FSD_SERVER"))
		fsdServerStruct := fsdServer.(*common.FSDServer)
		return fsdServerStruct

	}

	// Sort slice of servers by distance from request
	sort.Slice(initialServers, func(i, j int) bool {
		return initialServers[i].Distance < initialServers[j].Distance
	})

	// Get country for first server to be returned based upon distance
	// and populate a new slice with other servers in that country
	firstServer := initialServers[0].Country
	for _, server := range initialServers {
		if server.Country == firstServer {
			finalServers = append(finalServers, server)
		}
	}

	// Sort slice by remaining slots
	sort.Slice(finalServers, func(i, j int) bool {
		return finalServers[i].RemainingSlots > finalServers[j].RemainingSlots
	})

	// If no servers in the final slice return random otherwise return first element of finalServers
	// Value returned from finalServers should be the closest server to a user with the most available slots
	if len(finalServers) == 0 {
		logger.Error("No server found for request returning random")
		fsdServer, _ := fsdServers.Load(initialServers[0].Name)
		fsdServerStruct := fsdServer.(*common.FSDServer)
		fsdServerStruct.RemainingSlots -= 1
		return fsdServerStruct
	} else {
		fsdServer, _ := fsdServers.Load(finalServers[0].Name)
		fsdServerStruct := fsdServer.(*common.FSDServer)
		fsdServerStruct.RemainingSlots -= 1
		return fsdServerStruct
	}
}

func parseQuery(m *dns.Msg, sourceIp net.Addr) {
	dnsRateCounter.Incr(1)
	sourceIpParsed := net.IP{}
	if dnsIpOverride == "" {
		sourceIpParsed = net.ParseIP(strings.Split(sourceIp.String(), ":")[0])
	} else {
		sourceIpParsed = net.ParseIP(dnsIpOverride)
	}
	record, err := db.City(sourceIpParsed)
	if err != nil {
		log.Panic(err)
	}

	sourceIpLatLng := geodist.Coord{Lat: record.Location.Latitude, Lon: record.Location.Longitude}
	server := pickServerToReturn(sourceIpLatLng)

	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			rr, err := dns.NewRR(fmt.Sprintf("%s %s IN A %s", q.Name, viper.GetString("DNS_TTL"), server.IpAddress))
			if err == nil {
				m.Answer = append(m.Answer, rr)
			}
			logger.Info(fmt.Sprintf("IP: %s Served: %s", sourceIpParsed.String(), server.Name))
		}
	}
}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		parseQuery(m, w.RemoteAddr())
	}
	err := w.WriteMsg(m)
	if err != nil {
		return
	}
}

func startDnsServer() {
	_rpsCounter := ratecounter.NewRateCounter(1 * time.Second)
	dnsRateCounter = _rpsCounter
	rateCollector := newRateCollector()
	prometheus.MustRegister(rateCollector)
	geoip2DB, err := geoip2.Open("GeoLite2-City.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	db = geoip2DB

	dns.HandleFunc(viper.GetString("HOSTNAME_TO_SERVE"), handleDnsRequest)
	go func() {
		// Starts UDP DNS server
		serverUDP := &dns.Server{Addr: fmt.Sprintf(":%s", viper.GetString("DNS_PORT")), Net: "udp"}
		logger.Info(fmt.Sprintf("Starting UDP DNS server on port %s", viper.GetString("DNS_PORT")))
		err := serverUDP.ListenAndServe()
		defer func(server *dns.Server) {
			err := server.Shutdown()
			if err != nil {

			}
		}(serverUDP)
		if err != nil {
			logger.Fatal(fmt.Sprintf("Failed to start server: %s", err.Error()))
		}
	}()
	go func() {
		// Starts TCP DNS server
		serverTCP := &dns.Server{Addr: fmt.Sprintf(":%s", viper.GetString("DNS_PORT")), Net: "tcp"}
		logger.Info(fmt.Sprintf("Starting TCP DNS server on port %s", viper.GetString("DNS_PORT")))
		err := serverTCP.ListenAndServe()
		defer func(server *dns.Server) {
			err := server.Shutdown()
			if err != nil {

			}
		}(serverTCP)
		if err != nil {
			logger.Fatal(fmt.Sprintf("Failed to start server: %s", err.Error()))
		}
	}()
	logger.Info(fmt.Sprintf("Default FSD server returned %s", viper.GetString("DEFAULT_FSD_SERVER")))
}

func handleWebRequests() {
	logger.Info(fmt.Sprintf("Starting data web server at port %s", viper.GetString("HTTP_DATA_PORT")))

	// This is for submitting data during testing
	http.HandleFunc("/submit_data", func(w http.ResponseWriter, r *http.Request) {
		fsdServerJson := &common.FSDServer{}
		err := json.NewDecoder(r.Body).Decode(&fsdServerJson)
		if err != nil {
			return
		}
		fsdServer := common.NewMockFSDServer(fsdServerJson)
		fsdServers.Store(fsdServerJson.Name, fsdServer)
		logger.Info(fmt.Sprintf("%s | %d | %d | %d", fsdServer.Name, fsdServer.CurrentUsers, fsdServer.MaxUsers, fsdServer.AcceptingConnections()))
		w.Write([]byte(fmt.Sprintf("Updated server %s", fsdServerJson.Name)))
	})
	// Allows setting an IP override for DNS requests. Really only needed for testing.
	http.HandleFunc("/dns_ip_override", func(w http.ResponseWriter, r *http.Request) {
		httpBody, _ := io.ReadAll(r.Body)
		dnsIpOverride = string(httpBody)
		logger.Info(fmt.Sprintf("Set DNS IP override to %s", httpBody))
		w.Write([]byte(fmt.Sprintf("Set DNS IP override to %s", httpBody)))

	})

	if err := http.ListenAndServe(fmt.Sprintf(":%s", viper.GetString("HTTP_DATA_PORT")), nil); err != nil {
		log.Fatal(err)
	}
}
