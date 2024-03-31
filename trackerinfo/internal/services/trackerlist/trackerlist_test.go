package trackerlist_test

import (
	"context"
	"testing"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/logger/slogdiscard"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/services/trackerlist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStorage struct {
	data     []models.Tracker
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

func (ts *testStorage) List(ctx context.Context) ([]models.Tracker, error) {
	return ts.data, nil
}

type testFetcher struct {
	data []models.Tracker
}

func (tf *testFetcher) Fetch(ctx context.Context, out chan<- models.Tracker) error {
	for _, tr := range tf.data {
		out <- tr
	}
	return nil
}

func TestTrackerList_Update(t *testing.T) {

	testTracker1 := models.Tracker{
		OrigId:      "1",
		Description: "1",
		Latitude:    1,
		Longitude:   1,
	}

	testTracker1Up := models.Tracker{
		OrigId:      "1",
		Description: "1_up",
		Latitude:    1,
		Longitude:   1,
	}

	testTracker2 := models.Tracker{
		OrigId:      "2",
		Description: "2",
		Latitude:    2,
		Longitude:   2,
	}

	testTracker2Up := models.Tracker{
		OrigId:      "2",
		Description: "2_up",
		Latitude:    2,
		Longitude:   2,
	}

	t.Run("insert", func(t *testing.T) {
		t.Parallel()
		storage := &testStorage{}
		testTrackers := []models.Tracker{testTracker1, testTracker2}
		fetcher := &testFetcher{data: testTrackers}

		tl, err := trackerlist.New(slogdiscard.NewDiscardLogger(), storage)

		require.NoError(t, err)

		tl.AddSource(fetcher)
		tl.Update(context.Background())
		assert.Equal(t, len(testTrackers), storage.inserted, "should be equal")
		// fetching the same trackers. Nothing must change
		tl.Update(context.Background())
		assert.Equal(t, len(testTrackers), storage.inserted, "added insertions")
		assert.Equal(t, 0, storage.deleted, "deletions mustn't occur")
		assert.Equal(t, 0, storage.updated, "updates mustn't occur")

	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()
		testTrackers := []models.Tracker{testTracker1}
		storage := &testStorage{data: testTrackers}
		fetcher := &testFetcher{data: testTrackers}

		tl, err := trackerlist.New(slogdiscard.NewDiscardLogger(), storage)
		require.NoError(t, err)
		tl.AddSource(fetcher)
		tl.Update(context.Background())
		assert.Equal(t, 0, storage.inserted, "insertions mustn't occur")
		assert.Equal(t, 0, storage.deleted, "deletions mustn't occur")
		assert.Equal(t, 0, storage.updated, "updates mustn't occur")

		fetcher.data = []models.Tracker{testTracker1Up, testTracker2}
		tl.Update(context.Background())
		assert.Equal(t, 1, storage.inserted, "no insertion")
		assert.Equal(t, 0, storage.deleted, "deletions mustn't occur")
		assert.Equal(t, 1, storage.updated, "not updated")

		fetcher.data = []models.Tracker{testTracker1Up, testTracker2Up}
		tl.Update(context.Background())
		assert.Equal(t, 1, storage.inserted, "insertions mustn't occur")
		assert.Equal(t, 0, storage.deleted, "deletions mustn't occur")
		assert.Equal(t, 2, storage.updated, "not updated")

	})

	t.Run("delete", func(t *testing.T) {
		t.Parallel()
		storageTrackers := []models.Tracker{testTracker1, testTracker2}
		storage := &testStorage{data: storageTrackers}
		fetcherTrackers := []models.Tracker{testTracker2}
		fetcher := &testFetcher{data: fetcherTrackers}

		tl, err := trackerlist.New(slogdiscard.NewDiscardLogger(), storage)
		require.NoError(t, err)
		tl.AddSource(fetcher)
		tl.Update(context.Background())
		assert.Equal(t, 0, storage.inserted, "insertions mustn't occur")
		assert.Equal(t, 1, storage.deleted, "not deleted")
		assert.Equal(t, 0, storage.updated, "updates mustn't occur")

		fetcher.data = []models.Tracker{}
		tl.Update(context.Background())
		assert.Equal(t, 0, storage.inserted, "insertions mustn't occur")
		assert.Equal(t, 2, storage.deleted, "not deleted")
		assert.Equal(t, 0, storage.updated, "updates mustn't occur")

	})

}
