package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		Env     string `yaml:"env" env-default:"local"`
		Storage struct {
			Path string `yaml:"path" env-required:"true"`
		} `yaml:"storage"`
		GRPCServer struct {
			Port    int           `yaml:"port" env:"GRPC_SERVER_PORT" env-required:"true"`
			Timeout time.Duration `yaml:"timeout" env-default:"5s"`
		} `yaml:"grpc_server"`
		HTTPClient struct {
			Timeout time.Duration `yaml:"timeout" env-default:"10s"`
		} `yaml:"http_client"`
		Fetchers struct {
			UpdateInterval time.Duration `yaml:"update_interval" env-default:"10m"`
		} `yaml:"fetchers"`
		Tracing struct {
			Enabled     bool   `yaml:"enabled" env-default:"false"`
			OTLPGrpcURL string `yaml:"otlp_grpc_url" env:"OTLP_GRPC_URL"`
		} `yaml:"tracing"`
		Metrics struct {
			Enabled    bool `yaml:"enabled" env-default:"false"`
			HTTPServer struct {
				Port int `yaml:"port" env:"METRICS_HTTP_SERVER_PORT" env-default:"9090"`
			} `yaml:"http_server"`
		} `yaml:"metrics"`
	}
)

// gets config path from a command line flag, then from an env variable CONFIG_PATH
// then tries to load the config file
// panics if any errors occur
func MustLoad() *Config {

	path := getConfigPath()
	if len(path) == 0 {
		panic("config path is empty")
	}

	var config Config

	if err := cleanenv.ReadConfig(path, &config); err != nil {
		panic(fmt.Sprintf("config loading failed %v", err))
	}

	return &config
}

// gets a config path from a command line flag, then from an env variable CONFIG_PATH
// returns an empty string if there is nothing in mentioned earlier places
func getConfigPath() string {
	var path string

	flag.StringVar(&path, "config", "", "path to a config file")
	flag.Parse()

	if len(path) == 0 {
		path = os.Getenv("CONFIG_PATH")
	}

	return path
}
