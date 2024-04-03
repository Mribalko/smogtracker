package armaqi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
)

const (
	url        = "https://armaqi.org/api/waqi/list"
	sourceName = "armaqi"
)

type (
	Response struct {
		Trackers []Tracker `json:"stations"`
	}

	Tracker struct {
		Id       int    `json:"id"`
		Title    string `json:"title"`
		Position Position
	}

	Position struct {
		Latitude  float64 `json:"lat"`
		Longitude float64 `json:"lng"`
	}

	Armaqi struct {
		httpClient     *http.Client
		name           models.SourceName
		updateInterval time.Duration
	}
)

func New(client *http.Client, updateInterval time.Duration) *Armaqi {
	return &Armaqi{
		httpClient:     client,
		name:           sourceName,
		updateInterval: updateInterval,
	}
}

func (a *Armaqi) Name() models.SourceName {
	return a.name
}

func (a *Armaqi) UpdateInterval() time.Duration {
	return a.updateInterval
}

func (a *Armaqi) Fetch(ctx context.Context) ([]models.Tracker, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var decoded Response

	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	var res []models.Tracker

	for _, tracker := range decoded.Trackers {
		res = append(res, models.Tracker{
			OrigId:      fmt.Sprint(tracker.Id),
			Source:      sourceName,
			Description: tracker.Title,
			Latitude:    tracker.Position.Latitude,
			Longitude:   tracker.Position.Longitude,
		})
	}

	return res, nil
}
