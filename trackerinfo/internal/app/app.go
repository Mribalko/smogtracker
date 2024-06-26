package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/app/grpcapp"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/fetchers/armaqi"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/services/trackerlist"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/storage/sqlite"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type App struct {
	gRPCApp *grpcapp.App
	ctx     context.Context
	service *trackerlist.TrackerList
	log     *slog.Logger
}

func New(ctx context.Context,
	log *slog.Logger,
	tracer trace.Tracer,
	meter metric.Meter,
	httpTimeout time.Duration,
	updateInterval time.Duration,
	grpcPort int,
	storagePath string,
) (*App, error) {
	const op = "app.New"
	storage, err := sqlite.New(tracer, sqlite.WithStoragePath(storagePath))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	trackerListService, err := trackerlist.New(log, tracer, meter, storage)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	httpClient := &http.Client{Timeout: httpTimeout}

	armaqiFetcher := armaqi.New(httpClient, updateInterval)
	err = trackerListService.RegisterSource(armaqiFetcher)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	grpcApp := grpcapp.New(log, trackerListService, grpcPort)

	return &App{
		gRPCApp: grpcApp,
		ctx:     ctx,
		service: trackerListService,
		log:     log}, nil

}

func (a *App) Start() {
	a.service.StartUpdate(a.ctx)
	go a.gRPCApp.MustStart()
	a.log.Info("application started")

}

func (a *App) Shutdown(ctx context.Context) error {
	a.service.StopUpdate()
	a.gRPCApp.Stop()
	a.log.Info("application stopped")
	return nil
}
