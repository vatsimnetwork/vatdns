package dnshaiku

import (
	"fmt"
	"github.com/jftuga/geodist"
	"github.com/miekg/dns"
	"github.com/spf13/viper"
	"github.com/vatsimnetwork/vatdns/internal/logger"
	"log"
	"net"
)

func ParseQuery(m *dns.Msg, sourceIp net.Addr) {
	m.RecursionAvailable = false
	m.RecursionDesired = false
	m.Authoritative = true
	dnsRateCounter.Incr(1)
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			if q.Name != "fsd-http.connect.vatsim.net." {
				sourceIpParsed := net.IP{}
				if dnsIpOverride == "" {
					sourceIpParsed = IpToIpNET(sourceIp.String())
				} else {
					sourceIpParsed = net.ParseIP(dnsIpOverride)
				}
				record, err := db.City(sourceIpParsed)
				if err != nil {
					log.Panic(err)
				}
				sourceIpLatLng := geodist.Coord{Lat: record.Location.Latitude, Lon: record.Location.Longitude}
				server := PickServerToReturn(sourceIpLatLng)

				rr, err := dns.NewRR(fmt.Sprintf("%s %s IN A %s", q.Name, viper.GetString("DNS_TTL"), server.IpAddress))

				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
				logger.Info(fmt.Sprintf("DNS | IP: %s Served: %s", sourceIpParsed.String(), server.Name))
			} else {
				rr, err := dns.NewRR(fmt.Sprintf("%s %s IN A %s", q.Name, viper.GetString("DNS_TTL"), publicIp))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}

		case dns.TypeSOA:
			logger.Info("Served SOA record request")
			record := new(dns.SOA)
			// TODO: This should get turned into a variable or tags driven thing.
			record.Hdr = dns.RR_Header{Name: "connect.vatsim.net.", Rrtype: dns.TypeSOA, Class: dns.ClassINET, Ttl: 1}
			record.Ns = "prod-vatdns-hj146.server.vatsim.net."
			record.Mbox = "prod-vatdns-ad137.server.vatsim.net."
			record.Serial = 1
			record.Refresh = 3600
			record.Retry = 600
			record.Expire = 1209600
			record.Minttl = 1
			m.Answer = append(m.Answer, record)
		case dns.TypeNS:
			logger.Info("Served NS record request")
			// TODO: This should get turned into a variable or tags driven thing.
			vatdnsServers := []string{"prod-vatdns-hj146.server.vatsim.net", "prod-vatdns-ad137.server.vatsim.net"}
			for _, server := range vatdnsServers {
				// TODO: This should get turned into a variable or tags driven thing.
				rr, err := dns.NewRR(fmt.Sprintf("%s %s IN NS %s", "connect.vatsim.net", "60", server))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}

	}
}
