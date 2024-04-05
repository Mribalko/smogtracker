package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/app"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/config"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/logger"
)

func main() {
	cfg := config.MustLoad()

	log := logger.SetLogger(cfg.Env)

	ctx, _ := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	app, err := app.New(ctx, log, cfg.GRPC.Port, cfg.StoragePath)
	if err != nil {
		panic(err)
	}

	app.Start()

	<-ctx.Done()

	log.Info("Gracefully stopping service")
	app.Stop()
	log.Info("Gracefully stopped")
}
