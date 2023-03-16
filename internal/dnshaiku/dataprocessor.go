package dnshaiku

import (
	"context"
	"fmt"
	"github.com/digitalocean/godo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"strings"
	"time"
	"vatdns/internal/logger"
	"vatdns/pkg/common"
)

func dataProcessorManager() {
	enableFsdServerProm := make(chan string)
	go handleProm(enableFsdServerProm)
	ctx := context.TODO()
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}
	doClient := godo.NewFromToken(viper.GetString("DO_API_KEY"))
	go func() {
		for {
			if viper.GetBool("TEST_MODE") == false {
				logger.Debug("Checking tag for Droplets")
				droplets, _, _ := doClient.Droplets.ListByTag(ctx, viper.GetString("DO_TAG"), opt)
				logger.Debug("Checked tag for Droplets")
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
					go func(dName string) {
						fsdServerStruct.Polling(enableFsdServerProm)
					}(d.Name)
				}
				logger.Debug("Found all servers using tag, sleeping for a minute")
			} else {
				logger.Info("Running in test mode")
			}
			time.Sleep(60 * time.Second)
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
}

func handleProm(enableFsdServerProm chan string) {
	for fsdServer := range enableFsdServerProm {
		fsdServer, _ := fsdServers.Load(fsdServer)
		fsdServerStruct := fsdServer.(*common.FSDServer)
		fsdCollector := newFsdServersCollector(fsdServerStruct)
		prometheus.MustRegister(fsdCollector)
	}
}
