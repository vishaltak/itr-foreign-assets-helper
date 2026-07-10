package stock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// yahooChartBaseURL is the Yahoo Finance chart API endpoint.
const yahooChartBaseURL = "https://query2.finance.yahoo.com/v8/finance/chart"

// YahooClient fetches stock data from Yahoo Finance
type YahooClient struct {
	client  *http.Client
	cache   map[string][]PriceData // Simple cache to avoid repeated API calls
	baseURL string                 // configurable for testing
}

// NewYahooClient creates a new Yahoo Finance client
func NewYahooClient() *YahooClient {
	return &YahooClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:   make(map[string][]PriceData),
		baseURL: yahooChartBaseURL,
	}
}

// yahooResponse represents the JSON response from Yahoo Finance API.
// Prices are pointers so that a JSON null (a day with no data) is
// distinguishable from a genuine 0.0 and can be skipped.
type yahooResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []*float64 `json:"open"`
					High   []*float64 `json:"high"`
					Low    []*float64 `json:"low"`
					Close  []*float64 `json:"close"`
					Volume []*int64   `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// parseYahooChart decodes a Yahoo Finance chart response and returns the
// price series, skipping any day whose OHLC data is null.
func parseYahooChart(r io.Reader) ([]PriceData, error) {
	var data yahooResponse
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if data.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo API error: %s", data.Chart.Error.Description)
	}

	if len(data.Chart.Result) == 0 || len(data.Chart.Result[0].Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no data returned")
	}

	result := data.Chart.Result[0]
	quote := result.Indicators.Quote[0]

	var prices []PriceData
	for i := range result.Timestamp {
		// Skip days where the close (or open) is null - Yahoo returns null
		// for non-trading days and incomplete data.
		if i >= len(quote.Close) || i >= len(quote.Open) {
			continue
		}
		if quote.Close[i] == nil || quote.Open[i] == nil {
			continue
		}

		// A daily bar represents a trading date, but Yahoo's timestamp is the
		// intraday market-open instant. Normalize it to that calendar date at
		// UTC midnight so the rest of the pipeline compares/formats it as a
		// date, independent of the machine timezone. For US exchanges the open
		// (09:30 ET = 13:30/14:30 UTC) shares the trading date's UTC calendar
		// day, so the UTC date is the trading date.
		ts := time.Unix(result.Timestamp[i], 0).UTC()
		day := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, time.UTC)

		prices = append(prices, PriceData{
			Date:   day,
			Open:   deref(quote.Open, i),
			High:   deref(quote.High, i),
			Low:    deref(quote.Low, i),
			Close:  deref(quote.Close, i),
			Volume: derefInt(quote.Volume, i),
		})
	}

	return prices, nil
}

func deref(s []*float64, i int) float64 {
	if i < len(s) && s[i] != nil {
		return *s[i]
	}
	return 0
}

func derefInt(s []*int64, i int) int64 {
	if i < len(s) && s[i] != nil {
		return *s[i]
	}
	return 0
}

// FetchHistoricalData retrieves historical stock data
func (y *YahooClient) FetchHistoricalData(ticker string, startDate, endDate time.Time) ([]PriceData, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s-%s-%s", ticker, startDate.Format(time.DateOnly), endDate.Format(time.DateOnly))
	if cached, ok := y.cache[cacheKey]; ok {
		return cached, nil
	}

	// IMPORTANT: Yahoo Finance API treats period2 as exclusive
	// To include the end date, we need to add 1 day to it
	// Also ensure we're using the start of day for period1 and end of day for period2

	// Set to start of day for start date
	apiStartDate := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)

	// Add 1 day to end date to make it inclusive, and set to start of that next day
	apiEndDate := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)

	url := fmt.Sprintf(
		"%s/%s?period1=%d&period2=%d&interval=1d&includePrePost=false",
		y.baseURL, ticker, apiStartDate.Unix(), apiEndDate.Unix(),
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Yahoo requires a user agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching data: %w", err)
	}
	defer resp.Body.Close()

	// Read the whole body: Yahoo returns a useful error description in the
	// JSON body even for non-200 responses (e.g. 404 for a delisted symbol),
	// so prefer that message; otherwise fail loud on the status code.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	prices, err := parseYahooChart(bytes.NewReader(body))
	if err != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("yahoo request for %s failed with status %d: %w", ticker, resp.StatusCode, err)
		}
		return nil, err
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("no data returned for range %s to %s", startDate.Format(time.DateOnly), endDate.Format(time.DateOnly))
	}

	// Cache the result
	y.cache[cacheKey] = prices

	return prices, nil
}

// GetPeakClosingValue finds the maximum closing value in a date range
func (y *YahooClient) GetPeakClosingValue(ticker string, startDate, endDate time.Time) (float64, time.Time, error) {
	prices, err := y.FetchHistoricalData(ticker, startDate, endDate)
	if err != nil {
		return 0, time.Time{}, err
	}

	if len(prices) == 0 {
		return 0, time.Time{}, fmt.Errorf("no price data available")
	}

	maxClose := prices[0].Close
	maxDate := prices[0].Date

	for _, price := range prices[1:] {
		if price.Close > maxClose {
			maxClose = price.Close
			maxDate = price.Date
		}
	}

	return maxClose, maxDate, nil
}

// GetClosingPriceOnDate gets the closing price on or before a specific date
func (y *YahooClient) GetClosingPriceOnDate(ticker string, targetDate time.Time) (float64, time.Time, error) {
	// Fetch data for 10 days before to ensure we get data even if market was closed
	startDate := targetDate.AddDate(0, 0, -10)
	prices, err := y.FetchHistoricalData(ticker, startDate, targetDate)
	if err != nil {
		return 0, time.Time{}, err
	}

	if len(prices) == 0 {
		return 0, time.Time{}, fmt.Errorf("no price data available")
	}

	// Sort by date to ensure we get the latest
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Date.Before(prices[j].Date)
	})

	// Find the last trading day on or before target date
	var lastPrice PriceData
	for _, price := range prices {
		if price.Date.After(targetDate) {
			break
		}
		lastPrice = price
	}

	if lastPrice.Date.IsZero() {
		return 0, time.Time{}, fmt.Errorf("no trading data on or before %s", targetDate.Format("2006-01-02"))
	}

	return lastPrice.Close, lastPrice.Date, nil
}
