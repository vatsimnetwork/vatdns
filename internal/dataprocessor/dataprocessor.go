package dataprocessor

import (
	"context"
	"fmt"
	"github.com/digitalocean/godo"
	"github.com/go-yaml/yaml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"vatdns/internal/logger"
	"vatdns/pkg/common"
)

func Main() {
	logger.Info("Starting data processors")
	dataProcessorManager()
}

var (
	fsdServers sync.Map
)

func dataProcessorManager() {
	ctx := context.TODO()
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}
	doClient := godo.NewFromToken(viper.GetString("DO_API_KEY"))
	go func() {
		for {
			if viper.GetBool("TEST_MODE") == false {
				logger.Info("Checking tag for Droplets")
				droplets, _, _ := doClient.Droplets.ListByTag(ctx, viper.GetString("DO_TAG"), opt)
				logger.Info("Checked tag for Droplets")
				for _, d := range droplets {
					_, fsdInMap := fsdServers.Load(d.Name)
					if fsdInMap {
						continue
					}
					if strings.Contains(d.Name, "hub") {
						continue
					}
					if strings.Contains(d.Name, "sweatbox") {
						continue
					}
					logger.Info(fmt.Sprintf("Found fsd server %s | %s", d.Name, d.Networks.V4[1].IPAddress))
					fsdServers.Store(d.Name, common.NewFSDServer(&d))
					fsdServer, _ := fsdServers.Load(d.Name)
					fsdServerStruct := fsdServer.(*common.FSDServer)
					fsdCollector := newFsdServersCollector(fsdServerStruct)
					prometheus.MustRegister(fsdCollector)
					go func(dName string) {
						fsdServerStruct.Polling()
					}(d.Name)
				}
				logger.Info("Found all servers using tag, sleeping for a minute")
				time.Sleep(60 * time.Second)
			} else {
				logger.Info("Running in test mode")
				yamlData, err := os.ReadFile("testing.yaml")
				if err != nil {
					logger.Fatal("Unable to read testing.yaml")
				}
				testingData := common.TestingDataYaml{}
				_ = yaml.Unmarshal(yamlData, &testingData)
				for _, v := range testingData.MockFsdServers {
					_, fsdInMap := fsdServers.Load(v.Name)
					if fsdInMap {
						continue
					}
					fsdServers.Store(v.Name, common.NewMockFSDServer(&v))
					fsdServer, _ := fsdServers.Load(v.Name)
					fsdServerStruct := fsdServer.(*common.FSDServer)
					fsdCollector := newFsdServersCollector(fsdServerStruct)
					prometheus.MustRegister(fsdCollector)
					go func(dName string) {
						fsdServerStruct.Polling()
					}(v.Name)
					logger.Info(fmt.Sprintf("Found fsd server %s | %s", v.Name, v.IpAddress))
				}
			}
		}
	}()

	if viper.GetBool("ENABLE_CLOUDFLARE") {
		go func() {
			logger.Info("Cloudflare not fully implemented")
			//api, err := cloudflare.NewWithAPIToken(viper.GetString("CLOUDFLARE_API_KEY"))
			//if err != nil {
			//	log.Fatal(err)
			//}
			//ctx := context.Background()
			//rcZone := &cloudflare.ResourceContainer{
			//	Level:      cloudflare.ZoneRouteLevel,
			//	Identifier: viper.GetString("CLOUDFLARE_ZONE_ID"),
			//}
			//for {
			//	cfLb, err := api.GetLoadBalancer(ctx, rcZone, viper.GetString("CLOUDFLARE_LB_ID"))
			//	if err != nil {
			//		logger.Fatal(fmt.Sprintf("%s", err))
			//	}
			//	for _, pool := range cfLb.DefaultPools {
			//		rcPool := &cloudflare.ResourceContainer{
			//			Level:      cloudflare.AccountRouteLevel,
			//			Identifier: viper.GetString("CLOUDFLARE_ACCOUNT_ID"),
			//		}
			//		cfPool, err := api.GetLoadBalancerPool(ctx, rcPool, pool)
			//		if err != nil {
			//			logger.Error(fmt.Sprintf("%s", err))
			//			continue
			//		}
			//		api.UpdateLoadBalancerPool(ctx, rc, cloudflare.UpdateLoadBalancerPoolParams{LoadBalancer: cfPool})
			//	}
			//fsdServers.Range(func(k, v interface{}) bool {
			//	fsdServer, _ := fsdServers.Load(k)
			//	fsdServerStruct := fsdServer.(*common.FSDServer)
			//	logger.Info(fmt.Sprintf("Sending %s data to Cloudflare | %d", fsdServerStruct.Name, fsdServerStruct.AcceptingConnections))

			//	return true
			//})
			//	time.Sleep(5 * time.Second)
			//}
		}()
	}
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", viper.GetString("PROMETHEUS_METRICS_PORT")), nil))
}
