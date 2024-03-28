package armaqi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
		httpClient *http.Client
	}
)

func New(client *http.Client) *Armaqi {
	return &Armaqi{
		httpClient: client,
	}
}

func (a *Armaqi) Fetch(ctx context.Context, out chan<- models.Tracker) error {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var decoded Response

	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return err
	}

	for _, tracker := range decoded.Trackers {
		out <- models.Tracker{
			OrigId:      fmt.Sprint(tracker.Id),
			Source:      sourceName,
			Description: tracker.Title,
			Latitude:    tracker.Position.Latitude,
			Longitude:   tracker.Position.Longitude,
		}
	}

	return nil
}
