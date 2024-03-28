package trackerlist

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	sl "github.com/MRibalko/smogtracker/trackerinfo/internal/logger"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
)

type (
	Storage interface {
		Insert(ctx context.Context, tracker models.Tracker) error
		Update(ctx context.Context, tracker models.Tracker) error
		Delete(ctx context.Context, id models.Id) error
		List(ctx context.Context, source string) ([]models.Tracker, error)
	}

	Fetcher interface {
		Fetch(ctx context.Context, out chan<- models.Tracker) error
	}

	TrackerList struct {
		log     *slog.Logger
		storage Storage
		sources []Fetcher
		hashes  map[models.Id]models.Hash
	}
)

func New(logger *slog.Logger, storage Storage) (*TrackerList, error) {

	tl := &TrackerList{
		log:     logger,
		storage: storage,
		hashes:  make(map[models.Id]models.Hash),
	}

	trList, err := tl.storage.List(context.Background(), "")
	if err != nil {
		return nil, err
	}

	for _, tr := range trList {
		tl.hashes[tr.Id()] = tr.Hash()
	}

	return tl, nil
}

func (tl *TrackerList) AddSource(source Fetcher) {
	const op = "TrackerList.AddSource"
	log := tl.log.With(slog.String("op", op))
	log.Info("adding source ", source)
	tl.sources = append(tl.sources, source)
}

func (tl *TrackerList) Update(ctx context.Context) {
	const op = "TrackerList.Update"
	log := tl.log.With(slog.String("op", op))
	log.Info("Updating trackers")

	out := make(chan models.Tracker)
	updHashes := make(map[models.Id]models.Hash)
	var wg sync.WaitGroup

	for _, v := range tl.sources {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := v.Fetch(ctx, out); err != nil {
				log.Error("fetch failed", sl.Err(err))
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	for tr := range out {
		trHash, exist := tl.hashes[tr.Id()]

		if !exist {
			err := tl.storage.Insert(ctx, tr)
			if err != nil {
				log.Error("tracker insertion failed", slog.String("SourceId", string(tr.Id())), sl.Err(err))
				return
			}
		}

		if exist && strings.Compare(string(trHash), string(tr.Hash())) != 0 {
			err := tl.storage.Update(ctx, tr)
			if err != nil {
				log.Error("tracker update failed", slog.String("Id", string(tr.Id())), sl.Err(err))
				return
			}
		}
		updHashes[tr.Id()] = tr.Hash() // mark that the tracker exists in updated feed
		delete(tl.hashes, tr.Id())     // delete the tracker from stale hashes

	}

	for id := range tl.hashes {
		if err := tl.storage.Delete(ctx, id); err != nil {
			log.Error("tracker deletion failed", slog.String("Id", string(id)), sl.Err(err))
			return
		}
	}

	tl.hashes = updHashes

	log.Info("trackers updated")
}
