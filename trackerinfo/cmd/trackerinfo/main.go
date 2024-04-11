package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/app"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/config"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/logger"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/trace"
)

func main() {
	const serviceName = "smog/trackerinfo"
	cfg := config.MustLoad()

	log := logger.SetLogger(cfg.Env)

	tp, err := trace.New(cfg.Tracing.Enabled, serviceName, cfg.Env)
	if err != nil {
		panic(err)
	}

	tracer := tp.Tracer("")

	ctx, _ := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	app, err := app.New(ctx, log, tracer,
		cfg.HttpTimeout.Duration,
		cfg.FetchersUpdateInterval.Duration,
		cfg.GRPC.Port,
		cfg.StoragePath)
	if err != nil {
		panic(err)
	}

	app.Start()

	<-ctx.Done()

	log.Info("Gracefully stopping service")
	app.Stop()
	log.Info("Gracefully stopped")
}
