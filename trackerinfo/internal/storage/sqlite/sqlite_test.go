package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
	errStorage "github.com/MRibalko/smogtracker/trackerinfo/internal/storage"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/storage/sqlite"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestSqlite(t *testing.T) {
	const (
		storagePath   = "../../../storage/test_trackerinfo.db"
		migrationPath = "../../../migrations"
	)

	db, err := sql.Open("sqlite3", storagePath)
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
		os.Remove(storagePath)
	})

	storage, err := sqlite.New(storagePath)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Insert and read", func(t *testing.T) {
		t.Parallel()
		testTracker := models.Tracker{
			OrigId:      "1",
			Source:      "test",
			Description: "some description",
			Latitude:    1.3224,
			Longitude:   5.221,
		}

		err := storage.Insert(ctx, testTracker)
		require.NoError(t, err)

		err = storage.Insert(ctx, testTracker)
		require.ErrorIs(t, err, errStorage.ErrTrackerExists)

		res, err := storage.List(ctx, testTracker.Source)
		require.NoError(t, err)
		require.NotEmpty(t, res)

		require.Equal(t, testTracker, res[0])

		_, err = storage.List(ctx, "random")
		require.ErrorIs(t, err, errStorage.ErrSourceNotFound)

	})

	t.Run("Update", func(t *testing.T) {
		t.Parallel()
		testTracker := models.Tracker{
			OrigId:      "1",
			Source:      "update",
			Description: "some description",
			Latitude:    1.3224,
			Longitude:   5.221,
		}

		updTracker := testTracker
		updTracker.Description = "new"
		updTracker.Latitude = 2.1
		updTracker.Longitude = 1.1

		err := storage.Insert(ctx, testTracker)
		require.NoError(t, err)

		err = storage.Update(ctx, updTracker)
		require.NoError(t, err)

		res, err := storage.List(ctx, testTracker.Source)
		require.NoError(t, err)
		require.NotEmpty(t, res)

		require.Equal(t, updTracker, res[0])

	})

	t.Run("Delete", func(t *testing.T) {
		t.Parallel()
		testTracker := models.Tracker{
			OrigId:      "1",
			Source:      "delete",
			Description: "some description",
			Latitude:    1.3224,
			Longitude:   5.221,
		}

		err := storage.Insert(ctx, testTracker)
		require.NoError(t, err)

		err = storage.Delete(ctx, testTracker.Id())
		require.NoError(t, err)

		res, err := storage.List(ctx, testTracker.Source)
		require.Empty(t, res)
		require.ErrorIs(t, err, errStorage.ErrSourceNotFound)

	})

}
