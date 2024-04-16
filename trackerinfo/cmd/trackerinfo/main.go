package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/app"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/config"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/logger"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/metric"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/trace"
	"go.opentelemetry.io/otel"
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

	if cfg.Tracing.Enabled {
		err := trace.Init(
			ctx,
			trace.WithServiceName(serviceName),
			trace.WithServiceVersion(serviceVersion),
			trace.WithDeploymentEnv(cfg.Env),
			trace.WithOtelGrpcURL(cfg.Tracing.OTLPGrpcURL),
		)
		if err != nil {
			panic(fmt.Errorf("trace init failed %v", err))
		}
	}
	// if trace.Init wasn't called before, global tracing provider returns noop instance
	tracer := otel.GetTracerProvider().Tracer("") // using default service name

	if cfg.Metrics.Enabled {
		err := metric.Init(ctx,
			metric.WithServiceName(serviceName),
			metric.WithServiceVersion(serviceVersion),
			metric.WithDeploymentEnv(cfg.Env),
		)
		if err != nil {
			panic(fmt.Errorf("metrics init failed %v", err))
		}

		err = metric.StartServer(ctx, log, cfg.Metrics.HTTPServer.Port)
		if err != nil {
			panic(fmt.Errorf("metrics server start failed %v", err))
		}
	}
	// if metric.Init wasn't called before, global meter provider returns noop instance
	meter := otel.GetMeterProvider().Meter(serviceName)

	app, err := app.New(ctx, log, tracer, meter,
		cfg.HTTPClient.Timeout,
		cfg.Fetchers.UpdateInterval,
		cfg.GRPCServer.Port,
		cfg.Storage.Path)
	if err != nil {
		panic(err)
	}

	app.Start()

	<-ctx.Done()

	log.Info("Gracefully stopping service")
	app.Stop()
	log.Info("Gracefully stopped")
}
