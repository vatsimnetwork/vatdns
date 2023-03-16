package dnshaiku

import (
	"fmt"
	"github.com/miekg/dns"
	"github.com/oschwald/geoip2-golang"
	"github.com/paulbellamy/ratecounter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"log"
	"time"
	"vatdns/internal/logger"
)

func HandleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	for _, extra := range r.Extra {
		switch extra.(type) {
		case *dns.OPT:
			for _, o := range extra.(*dns.OPT).Option {
				switch e := o.(type) {
				case *dns.EDNS0_SUBNET:
					if e.Address != nil {
						logger.Info(fmt.Sprintf("edns subnet found %s/%d", e.Address, e.SourceNetmask))
					}
				}
			}
		}
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.SetEdns0(4096, true)
	m.Compress = false
	switch r.Opcode {
	case dns.OpcodeQuery:
		ParseQuery(m, w.RemoteAddr())
	}
	err := w.WriteMsg(m)
	if err != nil {
		return
	}
}

func StartDnsServer() {
	_rpsCounter := ratecounter.NewRateCounter(1 * time.Second)
	dnsRateCounter = _rpsCounter
	rateCollector := newRateCollector()
	prometheus.MustRegister(rateCollector)
	geoip2DB, err := geoip2.Open("GeoLite2-City.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	db = geoip2DB

	dns.HandleFunc("fsd.connect.vatsim.net", HandleDnsRequest)
	dns.HandleFunc("fsd-http.connect.vatsim.net", HandleDnsRequest)
	dns.HandleFunc("connect.vatsim.net", HandleDnsRequest)
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
