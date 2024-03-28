package storage

import "errors"

var (
	ErrTrackerExists  = errors.New("tracker already exists")
	ErrSourceNotFound = errors.New("source not found")
)
