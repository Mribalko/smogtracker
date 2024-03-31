package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/storage"
	"github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, err
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Insert(ctx context.Context, tracker models.Tracker) error {
	stmt, err := s.db.Prepare(`INSERT INTO 
								trackers(id, orig_id, source, description, latitude, longitude)
								VALUES(?, ?, ?, ?, ?, ?)`)
	if err != nil {
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
			return storage.ErrTrackerExists
		}
		return err
	}

	return nil

}

func (s *Storage) Update(ctx context.Context, tracker models.Tracker) error {
	stmt, err := s.db.Prepare(`UPDATE trackers
								SET description = ?, latitude = ?, longitude = ?
								WHERE id = ?`)
	if err != nil {
		return err
	}
	_, err = stmt.ExecContext(ctx,
		tracker.Description,
		tracker.Latitude,
		tracker.Longitude,
		tracker.Id())

	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) Delete(ctx context.Context, id models.Id) error {
	stmt, err := s.db.Prepare(`DELETE FROM trackers
								WHERE id = ?`)
	if err != nil {
		return err
	}

	_, err = stmt.ExecContext(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) List(ctx context.Context) ([]models.Tracker, error) {

	stmt, err := s.db.Prepare(`SELECT orig_id, source, description, latitude, longitude
								FROM trackers
								`)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrSourceNotFound
		}
		return nil, err
	}

	defer rows.Close()

	var res []models.Tracker

	for rows.Next() {
		tr := models.Tracker{}
		err := rows.Scan(&tr.OrigId, &tr.Source, &tr.Description, &tr.Latitude, &tr.Longitude)
		if err != nil {
			return nil, err
		}
		res = append(res, tr)
	}
	if len(res) == 0 {
		return nil, storage.ErrSourceNotFound
	}

	return res, nil
}
