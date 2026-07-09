package stock

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchHistoricalData_Non200StatusFailsLoud(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("upstream unavailable"))
	}))
	defer server.Close()

	client := NewYahooClient()
	client.baseURL = server.URL

	_, err := client.FetchHistoricalData(
		"AAPL",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 500")
}

func TestFetchHistoricalData_SurfacesAPIErrorOn404(t *testing.T) {
	// Yahoo returns the useful error description in the body even on 404.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"chart":{"result":null,"error":{"code":"Not Found","description":"No data found, symbol may be delisted"}}}`))
	}))
	defer server.Close()

	client := NewYahooClient()
	client.baseURL = server.URL

	_, err := client.FetchHistoricalData(
		"BADTICKER",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "No data found")
}

func TestParseYahooChart_SkipsNullPrices(t *testing.T) {
	// The middle day has null OHLC (a real Yahoo behaviour); it must be dropped
	// rather than treated as a 0.0 close.
	body := `{"chart":{"result":[{"timestamp":[1704067200,1704153600,1704240000],
		"indicators":{"quote":[{
			"open":[100.0,null,102.0],
			"high":[101.0,null,103.0],
			"low":[99.0,null,101.0],
			"close":[100.5,null,102.5],
			"volume":[10,null,30]
		}]}}],"error":null}}`

	prices, err := parseYahooChart(strings.NewReader(body))
	require.NoError(t, err)

	require.Len(t, prices, 2)
	assert.Equal(t, 100.5, prices[0].Close)
	assert.Equal(t, 102.5, prices[1].Close)
}

func TestParseYahooChart_SurfacesAPIError(t *testing.T) {
	body := `{"chart":{"result":[],"error":{"code":"Not Found","description":"No data found, symbol may be delisted"}}}`

	_, err := parseYahooChart(strings.NewReader(body))
	require.Error(t, err)
	require.Contains(t, err.Error(), "No data found")
}

// yahooTS returns the unix timestamp for noon UTC on the given date, matching
// how Yahoo returns intraday-anchored daily timestamps. Noon keeps the calendar
// date stable regardless of the test machine's timezone.
func yahooTS(year int, month time.Month, day int) int64 {
	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC).Unix()
}

// yahooBody builds a Yahoo chart API JSON response for the given series.
func yahooBody(t *testing.T, timestamps []int64, closes []float64) string {
	t.Helper()
	volumes := make([]int64, len(closes))
	resp := map[string]any{
		"chart": map[string]any{
			"result": []any{
				map[string]any{
					"timestamp": timestamps,
					"indicators": map[string]any{
						"quote": []any{
							map[string]any{
								"open":   closes,
								"high":   closes,
								"low":    closes,
								"close":  closes,
								"volume": volumes,
							},
						},
					},
				},
			},
			"error": nil,
		},
	}
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	return string(b)
}

// newYahooTestClient returns a client wired to an httptest server that serves
// the given series, plus a pointer to the request count.
func newYahooTestClient(t *testing.T, timestamps []int64, closes []float64) (*YahooClient, *int, *string) {
	t.Helper()
	var hits int
	var lastURL string
	body := yahooBody(t, timestamps, closes)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		lastURL = r.URL.String()
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)

	client := NewYahooClient()
	client.baseURL = server.URL
	return client, &hits, &lastURL
}

func TestYahooClient_FetchHistoricalData(t *testing.T) {
	timestamps := []int64{yahooTS(2024, 1, 2), yahooTS(2024, 1, 3), yahooTS(2024, 1, 4)}
	closes := []float64{185.0, 184.25, 181.9}
	client, _, lastURL := newYahooTestClient(t, timestamps, closes)

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC)

	data, err := client.FetchHistoricalData("AAPL", start, end)
	require.NoError(t, err)
	require.Len(t, data, 3)
	assert.Equal(t, 185.0, data[0].Close)
	assert.Equal(t, 181.9, data[2].Close)

	// The ticker is in the path and the end date is made inclusive (period2 = end + 1 day).
	assert.Contains(t, *lastURL, "/AAPL?")
	assert.Contains(t, *lastURL, fmt.Sprintf("period2=%d", end.AddDate(0, 0, 1).Unix()))
}

func TestYahooClient_FetchHistoricalData_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"chart":{"result":null,"error":{"code":"Not Found","description":"No data found, symbol may be delisted"}}}`)
	}))
	defer server.Close()

	client := NewYahooClient()
	client.baseURL = server.URL

	_, err := client.FetchHistoricalData("INVALIDTICKER123",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC))
	require.Error(t, err)
	require.Contains(t, err.Error(), "No data found, symbol may be delisted")
}

func TestYahooClient_GetPeakClosingValue(t *testing.T) {
	timestamps := []int64{yahooTS(2024, 1, 2), yahooTS(2024, 1, 3), yahooTS(2024, 1, 4)}
	closes := []float64{100.0, 250.0, 180.0} // peak on Jan 3
	client, _, _ := newYahooTestClient(t, timestamps, closes)

	maxClose, maxDate, err := client.GetPeakClosingValue("AAPL",
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	assert.Equal(t, 250.0, maxClose)
	assert.Equal(t, timestamps[1], maxDate.Unix())
}

func TestYahooClient_GetClosingPriceOnDate(t *testing.T) {
	// Target (31 Dec) is a non-trading day; the last trading day on or before
	// it is 29 Dec.
	timestamps := []int64{yahooTS(2023, 12, 27), yahooTS(2023, 12, 28), yahooTS(2023, 12, 29)}
	closes := []float64{170.0, 171.0, 172.0}
	client, _, _ := newYahooTestClient(t, timestamps, closes)

	targetDate := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	price, actualDate, err := client.GetClosingPriceOnDate("AAPL", targetDate)
	require.NoError(t, err)

	assert.Equal(t, 172.0, price)
	assert.Equal(t, timestamps[2], actualDate.Unix())
	assert.False(t, actualDate.After(targetDate), "actual date must be on or before target")
}

func TestYahooClient_Caching(t *testing.T) {
	timestamps := []int64{yahooTS(2024, 1, 2), yahooTS(2024, 1, 3)}
	closes := []float64{185.0, 184.25}
	client, hits, _ := newYahooTestClient(t, timestamps, closes)

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	data1, err := client.FetchHistoricalData("AAPL", start, end)
	require.NoError(t, err)

	data2, err := client.FetchHistoricalData("AAPL", start, end)
	require.NoError(t, err)

	assert.Equal(t, len(data1), len(data2))
	assert.Equal(t, 1, *hits, "second call should be served from cache")
}
