package main

import (
	"fmt"
	"github.com/spf13/viper"
	_ "net/http/pprof"
	"vatdns/internal/dnshaiku"
	"vatdns/internal/logger"
)

func main() {
	logger.Info("Reading config")
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	viper.SetDefault("PROMETHEUS_METRICS_PORT", "9102")
	viper.SetDefault("HTTP_DATA_PORT", "8080")
	viper.SetDefault("DNS_PORT", "10053")
	viper.SetDefault("DNS_TTL", "10")
	viper.SetDefault("TEST_MODE", false)
	viper.SetDefault("HOSTNAME_TO_SERVE", "fsd.connect.vatsim.net")
	viper.SetDefault("DEFAULT_FSD_SERVER", "")
	viper.SetDefault("SENTRY_DSN", "")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	logger.Info("dnshaiku - if people can't connect...it was DNS")
	dnshaiku.Main()
}
