package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/vatsimnetwork/vatdns/internal/dnshaiku"
	"github.com/vatsimnetwork/vatdns/internal/logger"
	_ "net/http/pprof"
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
	viper.SetDefault("HTTP_ENDPOINT_PORT", "8081")
	viper.SetDefault("FSD_SERVER_REMOVE_FAILURE_COUNT", 2)
	_ = viper.ReadInConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		logger.Info(fmt.Sprintf("Config file changed: %s", e.Name))
	})
	viper.WatchConfig()
	logger.Info("dnshaiku - if people can't connect...it was DNS")
	dnshaiku.Main()
}
