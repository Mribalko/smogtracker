package main

import (
	"github.com/MRibalko/smogtracker/trackerinfo/internal/config"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/logger"
)

func main() {
	cfg := config.MustLoad()
	_ = cfg

	log := logger.SetLogger(cfg.Env)
	_ = log
}
