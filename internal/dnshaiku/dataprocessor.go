package dnshaiku

import (
	"bufio"
	"context"
	"fmt"
	"github.com/digitalocean/godo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"github.com/vatsimnetwork/vatdns/internal/logger"
	"github.com/vatsimnetwork/vatdns/pkg/common"
	"net"
	"strings"
	"time"
)

func dataProcessorManager() {
	enableFsdServerProm := make(chan string)
	go handleProm(enableFsdServerProm)

	deregisterFsd := make(chan string)
	go handleFsdDeregister(deregisterFsd)

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
					if strings.Split(d.Name, ".")[0] != "fsd" {
						continue
					}
					dropletIp, err := d.PublicIPv4()
					if err != nil {
						logger.Error(fmt.Sprintf("Unable to find public IPv4 for FSD server %s, skipping", d.Name))
						continue
					}
					_, _, err = net.ParseCIDR(fmt.Sprintf("%s/32", dropletIp))
					if err != nil {
						logger.Error(fmt.Sprintf("Unable to validate IP for %s, skipping", d.Name))
						continue
					}
					logger.Info(fmt.Sprintf("Found FSD server %s | %s", d.Name, dropletIp))

					// Are we alive check
					c, err := net.Dial("tcp", fmt.Sprintf("%s:%s", dropletIp, "6809"))
					if err != nil {
						logger.Info(fmt.Sprintf("%s", err))
						continue
					}
					check, _ := bufio.NewReader(c).ReadString('\n')
					_ = c.Close()
					if strings.HasPrefix(check, "$DISERVER:CLIENT:VATSIM FSD") != true {
						logger.Info(fmt.Sprintf("FSD server %s failed initial health check, skipping", d.Name))
						continue
					} else {
						logger.Info(fmt.Sprintf("FSD server %s passed initial health check, starting polling", d.Name))
					}
					fsdServers.Store(d.Name, common.NewFSDServer(&d))
					fsdServer, _ := fsdServers.Load(d.Name)
					fsdServerStruct := fsdServer.(*common.FSDServer)
					go func(dName string) {
						fsdServerStruct.Polling(enableFsdServerProm, deregisterFsd)
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
		fsdServerStruct.PrometheusCollector = fsdCollector
		prometheus.MustRegister(fsdCollector)
	}
}

func handleFsdDeregister(deregisterFsd chan string) {
	for fsdServer := range deregisterFsd {
		fsdServerMap, fsdFound := fsdServers.Load(fsdServer)
		fsdServerStruct := fsdServerMap.(*common.FSDServer)
		if fsdFound {
			logger.Info(fmt.Sprintf("Found %s in fsd server list", fsdServer))
		} else {
			logger.Info(fmt.Sprintf("Failed to find %s in fsd server list", fsdServer))
			continue
		}
		prometheus.Unregister(fsdServerStruct.PrometheusCollector)
		fsdServers.Delete(fsdServer)
		_, fsdFound = fsdServers.Load(fsdServer)
		if fsdFound == false {
			logger.Info(fmt.Sprintf("Removed %s from fsd server list", fsdServer))
		} else {
			logger.Info(fmt.Sprintf("Failed to %s remove from fsd server list", fsdServer))
		}
	}
}
