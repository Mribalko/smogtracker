package shutdownlist

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/MRibalko/smogtracker/pkg/logger"
)

type (
	Shutdowner interface {
		Shutdown(context.Context) error
	}

	ShutdownList struct {
		log  *slog.Logger
		list []Shutdowner
	}
)

func New(log *slog.Logger) *ShutdownList {
	return &ShutdownList{
		log: log,
	}
}

func (sl *ShutdownList) Add(item Shutdowner) {
	sl.list = append(sl.list, item)
}

// Calls Shutdown() for all added list items
func (sl *ShutdownList) Shutdown(ctx context.Context) {
	for _, v := range sl.list {
		sl.log.Info("shutdown item", slog.String("type", reflect.TypeOf(v).String()))
		if err := v.Shutdown(ctx); err != nil {
			sl.log.Error("shutdown failed", logger.Err(err))
		}
	}

}
