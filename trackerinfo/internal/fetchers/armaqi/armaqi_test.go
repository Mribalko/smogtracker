package armaqi_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/MRibalko/smogtracker/trackerinfo/internal/fetchers/armaqi"
	"github.com/MRibalko/smogtracker/trackerinfo/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(f RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(f),
	}
}
func TestArmaqi_Fetch(t *testing.T) {

	const respJSON = `{
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

	testClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, "https://armaqi.org/api/waqi/list", req.URL.String())
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(respJSON)),
		}
	})

	armaqi := armaqi.New(testClient)
	out := make(chan models.Tracker, 2)
	defer close(out)

	err := armaqi.Fetch(context.Background(), out)
	require.NoError(t, err)

	require.Equal(t, models.Tracker{
		OrigId:      "76921",
		Source:      "armaqi",
		Description: "Kentron",
		Latitude:    40.182,
		Longitude:   44.516,
	}, <-out)

	require.Equal(t, models.Tracker{
		OrigId:      "397555",
		Source:      "armaqi",
		Description: "Nor Nork 2nd massive",
		Latitude:    40.2,
		Longitude:   44.582,
	}, <-out)

}
