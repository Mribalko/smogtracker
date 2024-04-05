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
)

type App struct {
	gRPCApp *grpcapp.App
	ctx     context.Context
	service *trackerlist.TrackerList
	log     *slog.Logger
}

func New(ctx context.Context, log *slog.Logger, grpcPort int, storagePath string) (*App, error) {
	const op = "app.New"
	storage, err := sqlite.New(storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	trackerListService, err := trackerlist.New(log, storage)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	armaqiFetcher := armaqi.New(&http.Client{Timeout: 10 * time.Second}, 1*time.Minute)
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
	a.log.Info("Application started")

}

func (a *App) Stop() {
	a.service.StopUpdate()
	a.gRPCApp.Stop()
	a.log.Info("Application stopped")
}
