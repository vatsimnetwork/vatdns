package dnshaiku

import (
	"encoding/json"
	"fmt"
	"github.com/jftuga/geodist"
	"github.com/miekg/dns"
	"github.com/oschwald/geoip2-golang"
	"github.com/paulbellamy/ratecounter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
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
	fsdServers            sync.Map
	promEnabledfsdServers sync.Map
	db                    *geoip2.Reader
	dnsRateCounter        *ratecounter.RateCounter
)

func Main() {
	// Starts a basic HTTP endpoint to get data from dataprocessor
	go handleWebRequests()
	// Starts a tcp+udp DNS server
	go startDnsServer()
	// Starts a Prometheus exporter
	go handleProm()
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", viper.GetString("PROMETHEUS_METRICS_PORT")), nil))
}

func pickServerToReturn(sourceIpLatLng geodist.Coord) common.FSDServer {
	// Slices are easier for sorting
	initialServers := make([]common.FSDServer, 0)

	// Get servers into a slice, skipping those that are not accepting connections
	fsdServers.Range(func(k, v interface{}) bool {
		fsdServerStruct := v.(common.FSDServer)
		if fsdServerStruct.AcceptingConnections == 0 {
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

	// Sort slice of servers by distance from request
	sort.Slice(initialServers, func(i, j int) bool {
		return initialServers[i].Distance < initialServers[j].Distance
	})

	// Get country for first server to be returned based upon distance
	// and populate a new slice with other servers in that country
	firstServer := initialServers[0].Country
	finalServers := make([]common.FSDServer, 0)
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
		return initialServers[0]
	} else {
		return finalServers[0]
	}
}

func parseQuery(m *dns.Msg, sourceIp net.Addr) {
	dnsRateCounter.Incr(1)
	sourceIpParsed := net.ParseIP(strings.Split(sourceIp.String(), ":")[0])
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
			logger.Info(fmt.Sprintf("IP: %s Served: %s Distance: %fmi", sourceIpParsed.String(), server.Name, server.Distance))
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
		// Starts udp DNS server
		serverUDP := &dns.Server{Addr: fmt.Sprintf(":%s", viper.GetString("DNS_PORT")), Net: "udp"}
		logger.Info(fmt.Sprintf("Starting at %s\n", viper.GetString("DNS_PORT")))
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
		// Starts tcp DNS server
		serverTCP := &dns.Server{Addr: fmt.Sprintf(":%s", viper.GetString("DNS_PORT")), Net: "tcp"}
		viper.GetString("DNS_PORT")
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
}

func handleWebRequests() {
	logger.Info(fmt.Sprintf("Starting data web server at port %s", viper.GetString("HTTP_DATA_PORT")))
	http.HandleFunc("/submit_data", func(w http.ResponseWriter, r *http.Request) {
		fsdServerJson := common.FSDServer{}
		err := json.NewDecoder(r.Body).Decode(&fsdServerJson)
		if err != nil {
			return
		}
		fsdServers.Store(fsdServerJson.Name, fsdServerJson)
		fsdServer, _ := fsdServers.Load(fsdServerJson.Name)
		fsdServerStruct := fsdServer.(common.FSDServer)
		logger.Info(fmt.Sprintf("%s | %d | %d | %d", fsdServerStruct.Name, fsdServerStruct.CurrentUsers, fsdServerStruct.MaxUsers, fsdServerStruct.AcceptingConnections))
		w.Write([]byte(fmt.Sprintf("Updated server %s", fsdServerJson.Name)))

	})
	if err := http.ListenAndServe(fmt.Sprintf(":%s", viper.GetString("HTTP_DATA_PORT")), nil); err != nil {
		log.Fatal(err)
	}
}

func handleProm() {
	for {
		fsdServers.Range(func(k, v interface{}) bool {
			fsdServerStruct := v.(common.FSDServer)
			if _, ok := promEnabledfsdServers.Load(fsdServerStruct.Name); ok == false {
				promEnabledfsdServers.Store(fsdServerStruct.Name, "")
				fsdCollector := newFsdServersCollector(fsdServerStruct)
				prometheus.MustRegister(fsdCollector)
			}
			return true
		})
		time.Sleep(5 * time.Second)
	}
}
