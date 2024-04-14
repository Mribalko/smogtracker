package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/storage"
	"github.com/mattn/go-sqlite3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type (
	Storage struct {
		db     *sql.DB
		tracer trace.Tracer
	}
	Option func(*Storage) error
)

func WithStoragePath(storagePath string) Option {
	return func(s *Storage) error {
		db, err := sql.Open("sqlite3", storagePath)
		if err != nil {
			return err
		}
		s.db = db
		return nil
	}
}

func WithDatabaseInstance(db *sql.DB) Option {
	return func(s *Storage) error {
		if err := db.Ping(); err != nil {
			return err
		}
		s.db = db
		return nil
	}
}

// Without any options creates memory sqlite
func New(tracer trace.Tracer, options ...Option) (*Storage, error) {

	storage := &Storage{tracer: tracer}

	for _, opt := range options {
		if err := opt(storage); err != nil {
			return nil, err
		}
	}

	if storage.db != nil {
		return storage, nil
	}

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	storage.db = db

	return storage, nil
}

func (s *Storage) Insert(ctx context.Context, tracker models.Tracker) error {
	const op = "sqlite.Insert"

	ctx, span := s.tracer.Start(ctx, op,
		trace.WithAttributes(attribute.String("trackerId", string(tracker.Id()))),
	)
	defer span.End()

	stmt, err := s.db.Prepare(`INSERT INTO 
								trackers(id, orig_id, source, description, latitude, longitude)
								VALUES(?, ?, ?, ?, ?, ?)`)
	if err != nil {
		span.SetStatus(codes.Error, "db error")
		return err
	}
	_, err = stmt.ExecContext(ctx,
		tracker.Id(),
		tracker.OrigId,
		tracker.Source,
		tracker.Description,
		tracker.Latitude,
		tracker.Longitude)

	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey {
			span.SetStatus(codes.Error, storage.ErrTrackerExists.Error())
			return storage.ErrTrackerExists
		}
		span.SetStatus(codes.Error, "db error")
		return err
	}

	return nil

}

func (s *Storage) Update(ctx context.Context, tracker models.Tracker) error {
	const op = "sqlite.Update"

	ctx, span := s.tracer.Start(ctx, op,
		trace.WithAttributes(attribute.String("trackerId", string(tracker.Id()))),
	)
	defer span.End()

	stmt, err := s.db.Prepare(`UPDATE trackers
								SET description = ?, latitude = ?, longitude = ?
								WHERE id = ?`)
	if err != nil {
		span.SetStatus(codes.Error, "db error")
		return err
	}
	_, err = stmt.ExecContext(ctx,
		tracker.Description,
		tracker.Latitude,
		tracker.Longitude,
		tracker.Id())

	if err != nil {
		span.SetStatus(codes.Error, "db error")
		return err
	}

	return nil
}

func (s *Storage) Delete(ctx context.Context, id models.Id) error {
	const op = "sqlite.Delete"

	ctx, span := s.tracer.Start(ctx, op,
		trace.WithAttributes(attribute.String("trackerId", string(id))),
	)
	defer span.End()

	stmt, err := s.db.Prepare(`DELETE FROM trackers
								WHERE id = ?`)
	if err != nil {
		span.SetStatus(codes.Error, "db error")
		return err
	}

	_, err = stmt.ExecContext(ctx, id)
	if err != nil {
		span.SetStatus(codes.Error, "db error")
		return err
	}

	return nil
}

func (s *Storage) Trackers(ctx context.Context) ([]models.Tracker, error) {
	const op = "sqlite.Trackers"
	ctx, span := s.tracer.Start(ctx, op)
	defer span.End()

	stmt, err := s.db.Prepare(`SELECT orig_id, source, description, latitude, longitude
								FROM trackers`)
	if err != nil {
		span.SetStatus(codes.Error, "db error")
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		span.SetStatus(codes.Error, "db error")
		return nil, err
	}
	defer rows.Close()

	var res []models.Tracker

	for rows.Next() {
		tr := models.Tracker{}
		err := rows.Scan(&tr.OrigId, &tr.Source, &tr.Description, &tr.Latitude, &tr.Longitude)
		if err != nil {
			span.SetStatus(codes.Error, "db error")
			return nil, err
		}
		res = append(res, tr)
	}

	span.SetAttributes(attribute.Int("trackers returned", len(res)))

	return res, nil
}

func (s *Storage) Sources(ctx context.Context) ([]string, error) {
	const op = "sqlite.Sources"
	ctx, span := s.tracer.Start(ctx, op)
	defer span.End()

	stmt, err := s.db.Prepare(`SELECT DISTINCT source
								FROM trackers`)
	if err != nil {
		span.SetStatus(codes.Error, "db error")
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		span.SetStatus(codes.Error, "db error")
		return nil, err
	}
	defer rows.Close()

	var res []string

	for rows.Next() {
		var source string
		err := rows.Scan(&source)

		if err != nil {
			span.SetStatus(codes.Error, "db error")
			return nil, err
		}

		res = append(res, source)
	}
	span.SetAttributes(attribute.Int("Sources returned", len(res)))
	return res, nil
}

func (s *Storage) IdsBySource(ctx context.Context, source string) ([]string, error) {
	const op = "sqlite.IdsBySource"
	ctx, span := s.tracer.Start(ctx, op,
		trace.WithAttributes(attribute.String("source", source)),
	)
	defer span.End()

	stmt, err := s.db.Prepare(`SELECT orig_id
								FROM trackers
								WHERE source = ?`)

	if err != nil {
		span.SetStatus(codes.Error, "db error")
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, source)
	if err != nil {
		span.SetStatus(codes.Error, "db error")
		return nil, err
	}
	defer rows.Close()

	var res []string

	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			span.SetStatus(codes.Error, "db error")
			return nil, err
		}

		res = append(res, id)
	}
	if len(res) == 0 {
		span.SetStatus(codes.Error, storage.ErrSourceNotFound.Error())
		return nil, storage.ErrSourceNotFound
	}

	span.SetAttributes(attribute.Int("Ids returned", len(res)))
	return res, nil
}
