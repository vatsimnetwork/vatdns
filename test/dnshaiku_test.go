package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-yaml/yaml"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
	"vatdns/internal/logger"
	"vatdns/pkg/common"
)

func dnsLookup() string {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, network, "127.0.0.1:10053")
		},
	}
	ip, _ := r.LookupHost(context.Background(), "fsd.connect.vatsim.net")
	return ip[0]
}

func TestDNS(t *testing.T) {
	os.Chdir("../test_data")
	testFiles, _ := os.ReadDir(".")
	for _, testFile := range testFiles {
		yamlData, err := os.ReadFile(testFile.Name())
		if err != nil {
			logger.Fatal(fmt.Sprintf("Unable to read %s", testFile.Name()))
		}
		testingDataFile := common.TestingDataYaml{}
		_ = yaml.Unmarshal(yamlData, &testingDataFile)

		for _, v := range testingDataFile.MockFsdServers {
			serverJson, _ := json.Marshal(v)
			requestURL := fmt.Sprintf("http://127.0.0.1:8080/submit_data")
			_, err := http.Post(requestURL, "application/json", bytes.NewBuffer(serverJson))
			if err != nil {
				logger.Error("Unable to submit data to DNS server")
			}
		}
		time.Sleep(1 * time.Second)
		for _, v := range testingDataFile.MockDnsQueries {
			body := []byte(v.SourceIpAddress)
			bodyReader := bytes.NewReader(body)

			requestURL := fmt.Sprintf("http://127.0.0.1:8080/dns_ip_override")
			_, _ = http.Post(requestURL, "", bodyReader)
			dnsResult := dnsLookup()
			assert.Equal(t, v.ExpectedIpReturned, dnsResult, "Should be the same")
		}

	}
}
