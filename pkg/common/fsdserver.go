package common

import (
	"fmt"
	"github.com/digitalocean/godo"
	"github.com/go-yaml/yaml"
	"github.com/prometheus/common/expfmt"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
	"vatdns/internal/logger"
)

type FSDServer struct {
	IpAddress      string  `json:"ip_address" yaml:"ip_address"`
	Name           string  `json:"name" yaml:"name"`
	Country        string  `json:"country" yaml:"country"`
	Latitude       float64 `json:"latitude" yaml:"latitude"`
	Longitude      float64 `json:"longitude" yaml:"longitude"`
	CurrentUsers   int     `json:"current_users" yaml:"current_users"`
	MaxUsers       int     `json:"max_users" yaml:"max_users"`
	RemainingSlots int     `json:"remaining_slots" yaml:"remaining_slots"`
	Distance       float64
}

func NewMockFSDServer(mockFsdServer *FSDServer) *FSDServer {
	possibleLocations := make(map[string]FSDServerLocation)
	possibleLocations["usa-w"] = FSDServerLocation{Latitude: 37.7749, Longitude: -122.431297}
	possibleLocations["usa-e"] = FSDServerLocation{Latitude: 40.7128, Longitude: -73.935242}
	possibleLocations["can"] = FSDServerLocation{Latitude: 43.6532, Longitude: -79.3832}
	possibleLocations["uk"] = FSDServerLocation{Latitude: 51.5072, Longitude: 0.1276}
	possibleLocations["ger"] = FSDServerLocation{Latitude: 50.1109, Longitude: 8.6821}
	possibleLocations["ams"] = FSDServerLocation{Latitude: 52.3676, Longitude: 4.9041}
	re := regexp.MustCompile("[^a-zA-Z-]")
	countryCode := re.ReplaceAllString(strings.Split(mockFsdServer.Name, ".")[1], "")

	return &FSDServer{
		Name:           mockFsdServer.Name,
		Country:        countryCode,
		IpAddress:      mockFsdServer.IpAddress,
		CurrentUsers:   mockFsdServer.CurrentUsers,
		MaxUsers:       mockFsdServer.MaxUsers,
		RemainingSlots: mockFsdServer.RemainingSlots,
		Latitude:       possibleLocations[countryCode].Latitude,
		Longitude:      possibleLocations[countryCode].Longitude,
		Distance:       0,
	}
}

func NewFSDServer(droplet *godo.Droplet) *FSDServer {
	possibleLocations := make(map[string]FSDServerLocation)
	possibleLocations["usa-w"] = FSDServerLocation{Latitude: 37.7749, Longitude: -122.431297}
	possibleLocations["usa-e"] = FSDServerLocation{Latitude: 40.7128, Longitude: -73.935242}
	possibleLocations["can"] = FSDServerLocation{Latitude: 43.6532, Longitude: -79.3832}
	possibleLocations["uk"] = FSDServerLocation{Latitude: 51.5072, Longitude: 0.1276}
	possibleLocations["ger"] = FSDServerLocation{Latitude: 50.1109, Longitude: 8.6821}
	possibleLocations["ams"] = FSDServerLocation{Latitude: 52.3676, Longitude: 4.9041}
	re := regexp.MustCompile("[^a-zA-Z-]")
	countryCode := re.ReplaceAllString(strings.Split(droplet.Name, ".")[1], "")
	publicIPv4, err := droplet.PublicIPv4()
	if err != nil {
		logger.Error(fmt.Sprintf("No IP address found for %s", droplet.Name))
	}

	return &FSDServer{
		Name:      droplet.Name,
		IpAddress: publicIPv4,
		Country:   countryCode,
		Latitude:  possibleLocations[countryCode].Latitude,
		Longitude: possibleLocations[countryCode].Longitude,
		Distance:  0,
	}
}
func (fsd *FSDServer) AcceptingConnections() int {
	if viper.GetInt("FSD_SLOT_BUFFER") > fsd.RemainingSlots {
		return 0
	} else {
		return 1
	}

}

func (fsd *FSDServer) Polling(enableFsdServerProm chan<- string) {
	enableFsdServerProm <- fsd.Name
	var parser expfmt.TextParser
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	ticker := time.NewTicker(time.Duration(viper.GetInt("FSD_SERVER_POLLING_INTERVAL")) * time.Second)
	for _ = range ticker.C {
		if viper.GetBool("TEST_MODE") == false {
			resp, err := client.Get(fmt.Sprintf("http://%s:9001/metrics", fsd.IpAddress))
			if err != nil {
				logger.Error(fmt.Sprintf("No response from request for %s", fsd.Name))
				continue
			}
			if err != nil {
				fmt.Println(err)
				fmt.Println("Decode failed")
			}
			promData, err := parser.TextToMetricFamilies(resp.Body)
			if err != nil {
				logger.Fatal(fmt.Sprintf("Bad prometheus data from FSD %s", fsd.Name))
			}
			for k, v := range promData {
				if k == "fsd_maxclients" {
					fsd.MaxUsers = int(*v.Metric[0].GetGauge().Value)
				}
				if k == "interface_client_current" {
					fsd.CurrentUsers = int(*v.Metric[0].GetGauge().Value)
				}
				if k == "fsd_remainingslots" {
					fsd.RemainingSlots = int(*v.Metric[0].GetGauge().Value)
				}
			}
			logger.Debug(fmt.Sprintf("Updated metrics for %s", fsd.Name))
		} else {
			testingData := TestingDataYaml{}
			yamlData, err := os.ReadFile("testing.yaml")
			if err != nil {
				logger.Fatal("Reading testing.yaml failed")
			}
			_ = yaml.Unmarshal(yamlData, &testingData)
			for _, v := range testingData.MockFsdServers {
				if v.Name == fsd.Name {
					fsd.MaxUsers = v.MaxUsers
					fsd.CurrentUsers = v.CurrentUsers
					fsd.RemainingSlots = v.RemainingSlots
				}
			}
		}

	}
}

type FSDServerLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"Longitude"`
}
