package models

import (
	"crypto/md5"
	"fmt"
)

type (
	Hash string
	Id   string

	Tracker struct {
		OrigId      string
		Source      string
		Description string
		Latitude    float64
		Longitude   float64
	}
)

// returns MD5 hash of fields Description, Latitude, Longitude
func (t *Tracker) Hash() Hash {

	h := md5.New()
	fmt.Fprintf(h, "%s|%f|%f", t.Description, t.Latitude, t.Longitude)
	return Hash(h.Sum(nil))
}

func (t *Tracker) Id() Id {
	return Id(fmt.Sprintf("%s|%s", t.Source, t.OrigId))
}
