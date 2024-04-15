package trackerlist

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	sl "github.com/MRibalko/smogtracker/trackerinfo/internal/logger"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type (
	Storage interface {
		Insert(ctx context.Context, tracker models.Tracker) error
		Update(ctx context.Context, tracker models.Tracker) error
		Delete(ctx context.Context, id models.Id) error
		Trackers(ctx context.Context) ([]models.Tracker, error)
		Sources(ctx context.Context) ([]string, error)
		IdsBySource(ctx context.Context, source string) ([]string, error)
	}

	Fetcher interface {
		Fetch(ctx context.Context) ([]models.Tracker, error)
		Name() models.SourceName
		UpdateInterval() time.Duration
	}

	TrackerList struct {
		log     *slog.Logger
		tracer  trace.Tracer
		metrics *instruments
		storage Storage
		sources map[models.SourceName]Fetcher
		mu      sync.Mutex
		hashes  map[models.SourceName]map[models.Id]models.Hash
		cancel  context.CancelFunc
		running bool
	}

	instruments struct {
		writeDbRequests metric.Int64Counter
		cacheRequests   metric.Int64Counter
	}
)

func New(logger *slog.Logger, tracer trace.Tracer, meter metric.Meter, storage Storage) (*TrackerList, error) {

	metrics, err := newInstruments(meter)
	if err != nil {
		return nil, err
	}

	tl := &TrackerList{
		log:     logger,
		tracer:  tracer,
		metrics: metrics,
		storage: storage,
		sources: make(map[models.SourceName]Fetcher),
		hashes:  make(map[models.SourceName]map[models.Id]models.Hash),
	}

	trList, err := tl.storage.Trackers(context.Background())
	if err != nil {
		return nil, err
	}

	for _, tr := range trList {
		if _, exists := tl.hashes[tr.SourceName()]; !exists {
			tl.hashes[tr.SourceName()] = make(map[models.Id]models.Hash)
		}

		tl.hashes[tr.SourceName()][tr.Id()] = tr.Hash()
	}

	return tl, nil
}

// Adds a new source to TrackerList
//
// Returns an error if the source had already been added
func (tl *TrackerList) RegisterSource(source Fetcher) error {
	const op = "TrackerList.RegisterSource"
	log := tl.log.With(slog.String("op", op))

	if len(source.Name()) == 0 {
		return fmt.Errorf("%s: name is empty", op)
	}

	if source.UpdateInterval() == 0 {
		return fmt.Errorf("%s: update interval must be positive", op)
	}

	if tl.running {
		return fmt.Errorf("%s: update is running", op)
	}

	if _, exists := tl.sources[source.Name()]; exists {
		return fmt.Errorf("%s: source already exists", op)
	}

	log.Info(fmt.Sprintf("adding source %s", source.Name()))
	tl.sources[source.Name()] = source
	if _, exists := tl.hashes[source.Name()]; !exists {
		tl.hashes[models.SourceName(source.Name())] = make(map[models.Id]models.Hash)
	}
	return nil
}

// Starts repeatable update of registered fetchers
func (tl *TrackerList) StartUpdate(ctx context.Context) {
	const op = "TrackerList.StartUpdate"

	log := tl.log.With(slog.String("op", op))
	log.Info("Trackers update started")

	updctx, cancel := context.WithCancel(ctx)
	tl.cancel = cancel
	tl.running = true

	var wg sync.WaitGroup

	for _, v := range tl.sources {
		wg.Add(1)
		go func() {
			log.Info(fmt.Sprintf("fetcher \"%s\" started, update interval %s", v.Name(), v.UpdateInterval()))
			defer wg.Done()
			t := time.NewTicker(v.UpdateInterval())
			defer t.Stop()

			for {

				res, err := v.Fetch(updctx)
				if err != nil {
					log.Error(fmt.Sprintf("fetch \"%s\" failed", v.Name()), sl.Err(err))
				}
				if err := tl.makeUpdates(updctx, v.Name(), res); err != nil {
					log.Error(fmt.Sprintf("update \"%s\" failed", v.Name()), sl.Err(err))
				}

				select {
				case <-updctx.Done():
					log.Info(fmt.Sprintf("fetcher \"%s\" stopped", v.Name()))
					return
				case <-t.C:

				}
			}
		}()
	}

	go func() {
		wg.Wait()
		tl.running = false
		log.Info("Trackers update stopped")
	}()

}

func (tl *TrackerList) StopUpdate() {
	const op = "TrackerList.StopUpdate"
	log := tl.log.With(slog.String("op", op))
	log.Info("Trackers update stopping")
	tl.cancel()
}

// Returns the list of added data sources
func (tl *TrackerList) Sources(ctx context.Context) ([]string, error) {
	const op = "TrackerList.Sources"
	_, span := tl.tracer.Start(ctx, op)
	defer span.End()

	if len(tl.sources) == 0 {
		span.SetStatus(codes.Error, "no sources")
		return nil, fmt.Errorf("%s: no sources", op)
	}

	var sources []string

	for k := range tl.sources {
		sources = append(sources, string(k))
	}
	span.SetAttributes(attribute.Int("sources returned", len(sources)))

	return sources, nil
}

// Return the list of tracker Ids from the source
func (tl *TrackerList) IdsBySource(ctx context.Context, source string) ([]string, error) {
	const op = "TrackerList.IdsBySource"
	ctx, span := tl.tracer.Start(ctx, op)
	defer span.End()

	if len(source) == 0 {
		span.SetStatus(codes.Error, "source string is empty")
		return nil, fmt.Errorf("%s: source string is empty", op)
	}

	ids, err := tl.storage.IdsBySource(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	span.SetAttributes(attribute.Int("ids returned", len(ids)))

	return ids, nil
}

// Returns the list of trackers from all added sources
func (tl *TrackerList) List(ctx context.Context) ([]models.Tracker, error) {
	const op = "TrackerList.List"
	ctx, span := tl.tracer.Start(ctx, op)
	defer span.End()

	list, err := tl.storage.Trackers(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	span.SetAttributes(attribute.Int("trackers returned", len(list)))

	return list, nil
}

func (tl *TrackerList) makeUpdates(ctx context.Context, source models.SourceName, updates []models.Tracker) error {
	const op = "TrackerList.makeUpdates"
	log := tl.log.With(slog.String("op", op))
	if len(updates) == 0 {
		return errors.New("updates slice is empty")
	}

	log.Info(fmt.Sprintf("Updating source %s", source))

	tl.mu.Lock()
	hashes, exists := tl.hashes[source]
	tl.mu.Unlock()
	if !exists {
		return fmt.Errorf("%s: no hash for source %s", op, source)
	}
	updHashes := make(map[models.Id]models.Hash)

	for _, tr := range updates {

		tl.metrics.cacheRequests.Add(ctx, 1)

		trHash, exist := hashes[tr.Id()]

		if !exist {
			tl.metrics.writeDbRequests.Add(ctx, 1)

			err := tl.storage.Insert(ctx, tr)
			if err != nil {
				log.Error("tracker insertion failed", slog.String("SourceId", string(tr.Id())), sl.Err(err))
				return err
			}
		}

		if exist && strings.Compare(string(trHash), string(tr.Hash())) != 0 {
			tl.metrics.writeDbRequests.Add(ctx, 1)

			err := tl.storage.Update(ctx, tr)
			if err != nil {
				log.Error("tracker update failed", slog.String("Id", string(tr.Id())), sl.Err(err))
				return err
			}
		}
		updHashes[tr.Id()] = tr.Hash() // mark that the tracker exists in updated feed
		delete(hashes, tr.Id())        // delete the tracker from stale hashes

	}

	for id := range hashes {
		if err := tl.storage.Delete(ctx, id); err != nil {
			log.Error("tracker deletion failed", slog.String("Id", string(id)), sl.Err(err))
			return err
		}
	}

	tl.mu.Lock()
	tl.hashes[source] = updHashes
	tl.mu.Unlock()

	log.Info("trackers updated")
	return nil
}

func newInstruments(meter metric.Meter) (*instruments, error) {
	writeDbRequests, err := meter.Int64Counter("writeDbRequests",
		metric.WithDescription("Number of write requests to db"),
		metric.WithUnit("{hit}"))
	if err != nil {
		return nil, err
	}

	cacheRequests, err := meter.Int64Counter("cacheRequests",
		metric.WithDescription("Number of requests to cache"),
		metric.WithUnit("{request}"))
	if err != nil {
		return nil, err
	}

	return &instruments{
		writeDbRequests: writeDbRequests,
		cacheRequests:   cacheRequests,
	}, nil

}
