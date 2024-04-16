package metric

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Creates and starts http prometheus server. Uses context to gracefully stop the server
func MustStartServer(ctx context.Context, log *slog.Logger, port int) *http.Server {
	const op = "metric.StartServer"
	log = log.With(slog.String("op", op))

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: promhttp.Handler(),
	}

	log.Info("starting server", slog.Int("port", port))

	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server start failed", err)
			panic(err)
		}

	}()
	return server
}
