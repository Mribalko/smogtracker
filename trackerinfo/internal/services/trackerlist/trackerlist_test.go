package trackerlist_test

import (
	"context"
	"testing"
	"time"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/logger/slogdiscard"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/services/trackerlist"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStorage struct {
	trackers []models.Tracker
	sources  []string
	ids      []string
	inserted int
	updated  int
	deleted  int
}

func (ts *testStorage) Insert(ctx context.Context, tracker models.Tracker) error {
	ts.inserted++
	return nil
}

func (ts *testStorage) Update(ctx context.Context, tracker models.Tracker) error {
	ts.updated++
	return nil
}

func (ts *testStorage) Delete(ctx context.Context, id models.Id) error {
	ts.deleted++
	return nil
}

func (ts *testStorage) Trackers(ctx context.Context) ([]models.Tracker, error) {
	return ts.trackers, nil
}

func (ts *testStorage) Sources(ctx context.Context) ([]string, error) {
	return ts.sources, nil
}

func (ts *testStorage) IdsBySource(ctx context.Context, source string) ([]string, error) {
	return ts.ids, nil
}

type testFetcher struct {
	data     []models.Tracker
	name     string
	interval time.Duration
}

func (tf *testFetcher) Fetch(ctx context.Context) ([]models.Tracker, error) {
	return tf.data, nil
}

func (tf *testFetcher) Name() models.SourceName {
	return models.SourceName(tf.name)
}

func (tf *testFetcher) UpdateInterval() time.Duration {
	return tf.interval
}

func TestTrackerList_RegisterSource(t *testing.T) {
	cases := []struct {
		name        string
		fetcher     trackerlist.Fetcher
		expectError bool
	}{
		{
			"no name",
			&testFetcher{
				name:     "",
				interval: 1,
			},
			true,
		},
		{
			"interval not correct",
			&testFetcher{
				name:     "test",
				interval: 0,
			},
			true,
		},
		{
			"correct input",
			&testFetcher{
				name:     "test",
				interval: 1,
			},
			false,
		},
		{
			"adding existing item",
			&testFetcher{
				name:     "test",
				interval: 1,
			},
			true,
		},
	}
	storage := &testStorage{}

	tl, err := newTrackerListWithStorage(t, storage)
	require.NoError(t, err)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := tl.RegisterSource(tt.fetcher); (err != nil) != tt.expectError {
				t.Error("expected error", err)
			}
		})
	}

	t.Run("can't register when running", func(t *testing.T) {
		tl.StartUpdate(context.Background())

		fetcher := testFetcher{name: "new", interval: 1}

		err := tl.RegisterSource(&fetcher)
		require.Error(t, err)

		tl.StopUpdate()
		time.Sleep(150 * time.Millisecond)
		err = tl.RegisterSource(&fetcher)

		require.NoError(t, err)

	})

}
func TestTrackerList_Update(t *testing.T) {

	testTracker1 := models.Tracker{
		OrigId:      "1",
		Source:      "source1",
		Description: "1",
		Latitude:    1,
		Longitude:   1,
	}

	testTracker1Up := models.Tracker{
		OrigId:      "1",
		Source:      "source1",
		Description: "1_up",
		Latitude:    1,
		Longitude:   1,
	}

	testTracker2 := models.Tracker{
		OrigId:      "2",
		Source:      "source2",
		Description: "2",
		Latitude:    2,
		Longitude:   2,
	}

	fetcher1 := &testFetcher{
		data:     []models.Tracker{testTracker1},
		name:     "source1",
		interval: 10 * time.Second,
	}

	fetcher2 := &testFetcher{
		data:     []models.Tracker{testTracker2},
		name:     "source2",
		interval: 10 * time.Second,
	}

	ctx := context.Background()

	t.Run("Insertion to empty storage", func(t *testing.T) {
		t.Parallel()
		storage := &testStorage{}

		tl, err := newTrackerListWithStorage(t, storage)
		require.NoError(t, err)

		err = tl.RegisterSource(fetcher1)
		require.NoError(t, err)

		tl.StartUpdate(ctx)
		time.Sleep(150 * time.Millisecond)
		tl.StopUpdate()
		assert.Equal(t, 1, storage.inserted, "1 insertion expected")
		assert.Equal(t, 0, storage.deleted, "no deletions expected")
		assert.Equal(t, 0, storage.updated, "no updates expected")

	})

	t.Run("Insertion existing item", func(t *testing.T) {
		t.Parallel()
		storage := &testStorage{trackers: []models.Tracker{testTracker1}}

		tl, err := newTrackerListWithStorage(t, storage)
		require.NoError(t, err)

		err = tl.RegisterSource(fetcher1)
		require.NoError(t, err)

		tl.StartUpdate(ctx)
		time.Sleep(150 * time.Millisecond)
		tl.StopUpdate()
		assert.Equal(t, 0, storage.inserted, "no insertions expected")
		assert.Equal(t, 0, storage.deleted, "no deletions expected")
		assert.Equal(t, 0, storage.updated, "no updates expected")
	})

	t.Run("Two fetchers empty storage", func(t *testing.T) {
		t.Parallel()
		storage := &testStorage{}

		tl, err := newTrackerListWithStorage(t, storage)
		require.NoError(t, err)

		err = tl.RegisterSource(fetcher1)
		require.NoError(t, err)

		err = tl.RegisterSource(fetcher2)
		require.NoError(t, err)

		tl.StartUpdate(ctx)
		time.Sleep(150 * time.Millisecond)
		tl.StopUpdate()
		assert.Equal(t, 2, storage.inserted, "2 insertions expected")
		assert.Equal(t, 0, storage.deleted, "no deletions expected")
		assert.Equal(t, 0, storage.updated, "no updates expected")
	})

	t.Run("Delete not existing", func(t *testing.T) {
		t.Parallel()
		storage := &testStorage{trackers: []models.Tracker{{
			OrigId:      "2",
			Source:      "source1",
			Description: "2",
			Latitude:    2,
			Longitude:   2,
		}}}

		tl, err := newTrackerListWithStorage(t, storage)
		require.NoError(t, err)

		err = tl.RegisterSource(fetcher1)
		require.NoError(t, err)

		tl.StartUpdate(ctx)
		time.Sleep(150 * time.Millisecond)
		tl.StopUpdate()
		assert.Equal(t, 1, storage.inserted, "1 insertion expected")
		assert.Equal(t, 1, storage.deleted, "1 deletion expected")
		assert.Equal(t, 0, storage.updated, "no updates expected")
	})
	t.Run("Update", func(t *testing.T) {
		storage := &testStorage{trackers: []models.Tracker{testTracker1Up}}

		tl, err := newTrackerListWithStorage(t, storage)
		require.NoError(t, err)

		err = tl.RegisterSource(fetcher1)
		require.NoError(t, err)

		tl.StartUpdate(ctx)
		time.Sleep(150 * time.Millisecond)
		tl.StopUpdate()
		assert.Equal(t, 0, storage.inserted, "no insertions expected")
		assert.Equal(t, 0, storage.deleted, "no deletions expected")
		assert.Equal(t, 1, storage.updated, "1 update expected")
	})

}

func newTrackerListWithStorage(t *testing.T, storage trackerlist.Storage) (*trackerlist.TrackerList, error) {
	t.Helper()
	tp, err := trace.New(false)
	if err != nil {
		return nil, err
	}

	tracer := tp.Tracer("")
	return trackerlist.New(slogdiscard.NewDiscardLogger(), tracer, storage)

}
