package config

import (
	"flag"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string        `json:"env" env-default:"local"`
	StoragePath string        `json:"storage_path" env-required:"true"`
	HttpTimeout time.Duration `json:"http_timeout" env-default:"1s"`
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

	return nil
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
