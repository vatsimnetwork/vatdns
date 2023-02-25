package dnshaiku

import (
	"encoding/json"
	"fmt"
	"github.com/jftuga/geodist"
	"github.com/spf13/viper"
	"io"
	"log"
	"net"
	"net/http"
	"vatdns/internal/logger"
	"vatdns/pkg/common"
)

func StartWebServer() {
	logger.Info(fmt.Sprintf("Starting IP endpoint server at port %s", viper.GetString("HTTP_ENDPOINT_PORT")))
	endpointHttp := http.NewServeMux()
	// This is for getting an IP to connect to using plain HTTP, no DOH
	endpointHttp.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		dnsRateCounter.Incr(1)
		sourceIpParsed := net.IP{}
		if dnsIpOverride == "" {
			sourceIpParsed = IpToIpNET(GetUserIPAddressHTTP(r))
		} else {
			sourceIpParsed = net.ParseIP(dnsIpOverride)
		}
		record, err := db.City(sourceIpParsed)
		if err != nil {
			log.Panic(err)
		}
		sourceIpLatLng := geodist.Coord{Lat: record.Location.Latitude, Lon: record.Location.Longitude}
		server := PickServerToReturn(sourceIpLatLng)
		logger.Info(fmt.Sprintf("HTTP | IP: %s Served: %s", sourceIpParsed.String(), server.Name))
		w.Write([]byte(server.IpAddress))
	})
	if err := http.ListenAndServe(fmt.Sprintf(":%s", viper.GetString("HTTP_ENDPOINT_PORT")), endpointHttp); err != nil {
		log.Fatal(err)
	}
}

func StartDataWebServer() {
	logger.Info(fmt.Sprintf("Starting data web server at port %s", viper.GetString("HTTP_DATA_PORT")))
	testingHttp := http.NewServeMux()
	// This is for submitting data during testing
	testingHttp.HandleFunc("/submit_data", func(w http.ResponseWriter, r *http.Request) {
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
	testingHttp.HandleFunc("/dns_ip_override", func(w http.ResponseWriter, r *http.Request) {
		httpBody, _ := io.ReadAll(r.Body)
		dnsIpOverride = string(httpBody)
		logger.Info(fmt.Sprintf("Set DNS IP override to %s", httpBody))
		w.Write([]byte(fmt.Sprintf("Set DNS IP override to %s", httpBody)))

	})
	if err := http.ListenAndServe(fmt.Sprintf(":%s", viper.GetString("HTTP_DATA_PORT")), testingHttp); err != nil {
		log.Fatal(err)
	}
}
