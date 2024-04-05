package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		Env         string     `json:"env" env-default:"local"`
		StoragePath string     `json:"storage_path" env-required:"true"`
		HttpTimeout Duration   `json:"http_timeout" env-default:"1s"`
		GRPC        GRPCConfig `json:"grpc"`
	}

	GRPCConfig struct {
		Port    int      `json:"port" env-required:"true"`
		Timeout Duration `json:"timeout" env-default:"5s"`
	}

	Duration struct {
		time.Duration
	}
)

func (d *Duration) UnmarshalJSON(b []byte) (err error) {

	sd := string(b[1 : len(b)-1])
	d.Duration, err = time.ParseDuration(sd)
	if err != nil {
		return err
	}
	return nil
}

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
		panic("config loading failed: " + err.Error())
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
