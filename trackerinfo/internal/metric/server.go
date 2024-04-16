package metric

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Creates and starts http prometheus server. Uses context to gracefully stop the server
func StartServer(ctx context.Context, log *slog.Logger, port int) error {
	const op = "metric.StartServer"
	log = log.With(slog.String("op", op))

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: promhttp.Handler(),
	}

	log.Info("starting server", slog.Int("port", port))
	go func() {
		<-ctx.Done()
		log.Info("stopping server", slog.Int("port", port))
		if err := server.Shutdown(ctx); err != nil {
			log.Error("server gracefull shutdown failed", err)
		}
	}()

	err := server.ListenAndServe()
	if err != nil {
		log.Error("server start failed", err)
		return err
	}

	return nil

}
