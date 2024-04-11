package app_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	trackerinfov1 "github.com/MRibalko/smogtracker/protos/gen/go"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/app"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/logger/slogdiscard"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/trace"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestApp_LoadAndDisplay(t *testing.T) {

	// fake default transport
	const (
		respJSON = `{
		"stations": [
		  {
			"id": 76921,
			"title": "Kentron",
			"position": {
			  "lat": 40.182,
			  "lng": 44.516
			},
			"aqi": 15
		  },
		  {
			"id": 397555,
			"title": "Nor Nork 2nd massive",
			"position": {
			  "lat": 40.2,
			  "lng": 44.582
			},
			"aqi": 9
		  }
		]
	  }`
		migrationPath = "../../migrations"
		storagePath   = "../../storage/testStorage.db"
		grpcPort      = 44443
		url           = "https://armaqi.org/api/waqi/list"
	)

	ctx := context.Background()
	http.DefaultTransport = RoundTripFunc(func(req *http.Request) *http.Response {
		var body io.ReadCloser

		if strings.Compare(req.URL.String(), url) == 0 {
			body = io.NopCloser(strings.NewReader(respJSON))
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}
	})

	// create test db
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
		os.Remove(storagePath)
	})

	tp, err := trace.New(false, "", "test")
	require.NoError(t, err)

	tracer := tp.Tracer("")

	// create and start test app
	app, err := app.New(ctx,
		slogdiscard.NewDiscardLogger(),
		tracer,
		10*time.Second,
		10*time.Minute,
		grpcPort,
		storagePath)
	require.NoError(t, err)

	app.Start()

	// create grpc client
	conn, err := grpc.DialContext(
		ctx, fmt.Sprintf("localhost:%d", grpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	require.NoError(t, err)

	grpcClient := trackerinfov1.NewTrackerInfoClient(conn)

	t.Run("List", func(t *testing.T) {
		t.Parallel()
		resp, err := grpcClient.List(ctx, &trackerinfov1.EmptyRequest{})
		require.NoError(t, err)

		want := []models.Tracker{
			{
				OrigId:      "76921",
				Source:      "armaqi",
				Description: "Kentron",
				Latitude:    40.182,
				Longitude:   44.516,
			},
			{
				OrigId:      "397555",
				Source:      "armaqi",
				Description: "Nor Nork 2nd massive",
				Latitude:    40.2,
				Longitude:   44.582,
			},
		}

		got := make([]models.Tracker, 0, len(resp.Result))

		for _, v := range resp.Result {
			got = append(got, models.Tracker{
				OrigId:      v.OrigId,
				Source:      v.Source,
				Description: v.Description,
				Latitude:    v.Latitude,
				Longitude:   v.Longitude,
			})
		}

		require.Equal(t, want, got)
	})

	t.Run("IdsBySource", func(t *testing.T) {
		res, err := grpcClient.IdsBySource(ctx, &trackerinfov1.SourceRequest{Source: "armaqi"})
		require.NoError(t, err)

		want := []string{"76921", "397555"}

		require.Equal(t, want, res.Result)
	})

	t.Run("Sources", func(t *testing.T) {
		res, err := grpcClient.Sources(ctx, &trackerinfov1.EmptyRequest{})
		require.NoError(t, err)

		want := []string{"armaqi"}

		require.Equal(t, want, res.Result)
	})

}
