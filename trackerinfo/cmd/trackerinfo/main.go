package main

import (
	"context"
	"fmt"
	"os/signal"
	"reflect"
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

type shutdown interface {
	Shutdown(context.Context) error
}

func main() {

	cfg := config.MustLoad()

	log := logger.SetLogger(cfg.Env)

	ctx, _ := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	var shutdownList []shutdown
	defer func() {
		for _, item := range shutdownList {
			log.Info("stopping: " + reflect.TypeOf(item).String())
			item.Shutdown(ctx)
		}
	}()

	if cfg.Tracing.Enabled {
		tp, err := trace.New(
			ctx,
			trace.WithServiceName(serviceName),
			trace.WithServiceVersion(serviceVersion),
			trace.WithDeploymentEnv(cfg.Env),
			trace.WithOtelGrpcURL(cfg.Tracing.OTLPGrpcURL),
		)
		if err != nil {
			panic(fmt.Errorf("trace init failed %v", err))
		}
		shutdownList = append(shutdownList, tp)
	}
	// if trace.Init wasn't called before, global tracing provider returns noop instance
	tracer := otel.GetTracerProvider().Tracer("") // using default service name

	if cfg.Metrics.Enabled {
		mp, err := metric.New(ctx,
			metric.WithServiceName(serviceName),
			metric.WithServiceVersion(serviceVersion),
			metric.WithDeploymentEnv(cfg.Env),
		)
		if err != nil {
			panic(fmt.Errorf("metrics init failed %v", err))
		}
		shutdownList = append(shutdownList, mp)

		metricsServer := metric.MustStartServer(ctx, log, cfg.Metrics.HTTPServer.Port)
		shutdownList = append(shutdownList, metricsServer)

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
