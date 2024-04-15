package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/app"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/config"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/logger"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/metric"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/trace"
)

var (
	serviceName    = "smog/trackerinfo"
	serviceVersion = "0.2.0"
)

func main() {

	cfg := config.MustLoad()

	log := logger.SetLogger(cfg.Env)

	ctx, _ := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	tracerProvider, err := trace.New(
		ctx,
		cfg.Tracing.Enabled,
		trace.WithServiceName(serviceName),
		trace.WithServiceVersion(serviceVersion),
		trace.WithDeploymentEnv(cfg.Env),
		trace.WithOtelGrpcURL(cfg.Tracing.OTLPGrpcURL),
	)
	if err != nil {
		panic(err)
	}

	tracer := tracerProvider.Tracer("") // using default service name

	meterProvider, err := metric.New(ctx,
		true,
		metric.WithServiceName(serviceName),
		metric.WithServiceVersion(serviceVersion),
		metric.WithDeploymentEnv(cfg.Env),
	)

	meter := meterProvider.Meter(serviceName)

	app, err := app.New(ctx, log, tracer, meter,
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
	tracerProvider.Shutdown(ctx)
	meterProvider.Shutdown(ctx)
	log.Info("Gracefully stopped")
}
