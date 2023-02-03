package main

import (
	"fmt"
	"github.com/spf13/viper"
	_ "net/http/pprof"
	"vatdns/internal/dataprocessor"
	"vatdns/internal/logger"
)

func main() {
	logger.Info("Reading config")
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	viper.SetDefault("PROMETHEUS_METRICS_PORT", "9101")
	viper.SetDefault("DO_API_KEY", "")
	viper.SetDefault("DO_TAG", nil)
	viper.SetDefault("FSD_SERVER_POLLING_INTERVAL", 5)
	viper.SetDefault("FSD_SLOT_BUFFER", 5)
	viper.SetDefault("ENABLE_CLOUDFLARE", false)
	viper.SetDefault("CLOUDFLARE_API_KEY", "")
	viper.SetDefault("CLOUDFLARE_LB_ID", "")
	viper.SetDefault("CLOUDFLARE_ZONE_ID", "")
	viper.SetDefault("CLOUDFLARE_ACCOUNT_ID", "")
	viper.SetDefault("TEST_MODE", false)

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	logger.Info("dataprocessor - Push and pull metrics for placing client connections on fsd")
	dataprocessor.Main()
}
