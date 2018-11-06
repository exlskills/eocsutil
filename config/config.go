package config

import (
	"fmt"

	"github.com/exlinc/golang-utils/envconfig"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// The app is in production or debug mode
	Mode                   string `envconfig:"MODE" default:"production"`
	MgoDBName              string `envconfig:"MGO_DB_NAME"`
	GHWebhookSecret        string `envconfig:"GH_WEBHOOK_SECRET"`
	GHServerAddr           string `envconfig:"GH_SERVER_ADDR" default:"0.0.0.0"`
	GHServerPort           string `envconfig:"GH_SERVER_PORT" default:"3344"`
	GHServerMongoURI       string `envconfig:"GH_SERVER_MONGO_URI"`
	ElasticsearchURI       string `envconfig:"ELASTICSEARCH_URI"`
	ElasticsearchBaseIndex string `envconfig:"ELASTICSEARCH_BASE_INDEX"`
}

var conf *Config

const (
	DebugMode      = "debug"
	ProductionMode = "production"
)

func init() {
	conf = &Config{}
	err := envconfig.Process("eocs_util", conf)
	if err != nil {
		fmt.Println("Fatal error processing configuration")
		panic(err)
	}
	l := conf.GetLogger()
	if !conf.IsDebugMode() && !conf.IsProductionMode() {
		l.Fatal("Invalid EOCS_UTIL variable, it must be either `debug` or `production`")
	}

}

// Cfg returns the configuration - will panic if the config has not been loaded or is nil (which shouldn't happen as that's implicit in the package init)
func Cfg() *Config {
	if conf == nil {
		panic("Config is nil")
	}
	return conf
}

func (cfg *Config) GetLogger() *logrus.Logger {
	var l = logrus.New()
	return l
}

func (cfg *Config) IsDebugMode() bool {
	return cfg.Mode == DebugMode
}

func (cfg *Config) IsProductionMode() bool {
	return cfg.Mode == ProductionMode
}
