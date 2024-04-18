package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
	errStorage "github.com/MRibalko/smogtracker/trackerinfo/internal/storage"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/storage/sqlite"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestSqlite(t *testing.T) {
	const (
		migrationPath = "../../../migrations"
	)

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	require.NoError(t, err)

	m, err := migrate.NewWithDatabaseInstance("file://"+migrationPath, "sqlite3", driver)
	require.NoError(t, err)

	if err = m.Up(); !errors.Is(err, migrate.ErrNoChange) {
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		m.Down()
	})

	ctx := context.Background()

	storage, err := sqlite.New(otel.Tracer(""), sqlite.WithDatabaseInstance(db))
	require.NoError(t, err)

	testTracker := models.Tracker{
		OrigId:      "1",
		Source:      "test1",
		Description: "some description",
		Latitude:    1.3224,
		Longitude:   5.221,
	}

	t.Run("Insert", func(t *testing.T) {

		err := storage.Insert(ctx, testTracker)
		require.NoError(t, err)

		err = storage.Insert(ctx, testTracker)
		require.ErrorIs(t, err, errStorage.ErrTrackerExists)

		res, err := storage.Trackers(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, res)

		require.Equal(t, testTracker, res[0])

	})

	t.Run("Update", func(t *testing.T) {

		updTracker := testTracker
		updTracker.Description = "new"
		updTracker.Latitude = 2.1
		updTracker.Longitude = 1.1

		err = storage.Update(ctx, updTracker)
		require.NoError(t, err)

		res, err := storage.Trackers(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, res)

		require.Equal(t, updTracker, res[0])

	})

	t.Run("Delete", func(t *testing.T) {

		err = storage.Delete(ctx, testTracker.Id())
		require.NoError(t, err)

		res, err := storage.Trackers(ctx)
		require.Empty(t, res)
		require.NoError(t, err)

	})

	t.Run("ModifiedTrackers", func(t *testing.T) {
		err := storage.Insert(ctx, testTracker)
		require.NoError(t, err)

		testTracker1 := testTracker
		testTracker1.OrigId = "2"

		err = storage.Insert(ctx, testTracker1)
		require.NoError(t, err)

		testTracker1.Description = "new description"
		time.Sleep(1 * time.Second)
		now := time.Now()
		err = storage.Update(ctx, testTracker1)
		require.NoError(t, err)

		res, err := storage.ModifiedTrackers(ctx, now)
		require.NoError(t, err)
		require.NotEmpty(t, res)

		require.Equal(t, testTracker1, res[0])
	})

	t.Run("Sources", func(t *testing.T) {
		sources := []string{"test1", "test2", "test3"}
		for _, source := range sources {
			err := storage.Insert(ctx, models.Tracker{
				OrigId: "1",
				Source: source,
			})
			require.NoError(t, err)
		}

		err := storage.Insert(ctx, models.Tracker{
			OrigId: "2",
			Source: sources[0],
		})
		require.NoError(t, err)

		got, err := storage.Sources(ctx)
		require.NoError(t, err)
		require.Equal(t, sources, got)
		t.Cleanup(func() {
			m.Down()
			m.Up()
		})
	})

	t.Run("IdsBySource", func(t *testing.T) {
		const (
			source            = "source1"
			notExistingSource = "nosource"
		)
		ids := []string{"id1", "id2"}
		for _, id := range ids {
			err := storage.Insert(ctx, models.Tracker{
				OrigId: id,
				Source: source,
			})
			require.NoError(t, err)
		}

		res, err := storage.IdsBySource(ctx, notExistingSource)
		require.ErrorIs(t, err, errStorage.ErrSourceNotFound)
		require.Empty(t, res)

		res, err = storage.IdsBySource(ctx, source)
		require.NoError(t, err)
		require.Equal(t, ids, res)

	})

}
