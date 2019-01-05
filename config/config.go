package config

import (
	"fmt"
	"os"

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
	GHUserToken            string `envconfig:"GH_USER_TOKEN"`
	GHAutoGenCommitMsg     string `envconfig:"GH_AUTOGEN_COMMIT_MSG" default:"auto#gen"`
	ElasticsearchURI       string `envconfig:"ELASTICSEARCH_URI"`
	ElasticsearchBaseIndex string `envconfig:"ELASTICSEARCH_BASE_INDEX" default:"learn"`
	SMTPFromName           string `envconfig:"SMTP_FROM_NAME" default:"EOCS Course Loader Service"`
	SMTPFromAddress        string `envconfig:"SMTP_FROM_ADDRESS" default:"noreply@exlskills.com"`
	SMTPHost               string `envconfig:"SMTP_HOST" default:"smtp.sendgrid.net"`
	SMTPConnectionString   string `envconfig:"SMTP_CONNECTION_STRING" default:"smtp.sendgrid.net:587"`
	SMTPUserName           string `envconfig:"SMTP_USER_NAME" default:"apikey"`
	SMTPPassword           string `envconfig:"SMTP_PASSWORD"`
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
	//var l = logrus.New()
	logLvl := logrus.InfoLevel
	if cfg.IsDebugMode() {
		logLvl = logrus.DebugLevel
	}
	var l = &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logLvl,
	}
	return l
}

func (cfg *Config) IsDebugMode() bool {
	return cfg.Mode == DebugMode
}

func (cfg *Config) IsProductionMode() bool {
	return cfg.Mode == ProductionMode
}
