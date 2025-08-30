package stock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// YahooClient fetches stock data from Yahoo Finance
type YahooClient struct {
	client *http.Client
	cache  map[string][]PriceData // Simple cache to avoid repeated API calls
}

// NewYahooClient creates a new Yahoo Finance client
func NewYahooClient() *YahooClient {
	return &YahooClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: make(map[string][]PriceData),
	}
}

// yahooResponse represents the JSON response from Yahoo Finance API
type yahooResponse struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
					Close  []float64 `json:"close"`
					Volume []int64   `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
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
		"https://query2.finance.yahoo.com/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d&includePrePost=false",
		ticker, apiStartDate.Unix(), apiEndDate.Unix(),
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

	//if resp.StatusCode != http.StatusOK {
	//	return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	//}

	var data yahooResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if data.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo API error: %s", data.Chart.Error.Description)
	}

	if len(data.Chart.Result) == 0 || len(data.Chart.Result[0].Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no data returned for ticker %s", ticker)
	}

	result := data.Chart.Result[0]
	quote := result.Indicators.Quote[0]

	var prices []PriceData
	for i := range result.Timestamp {
		// Skip if any values are null (happens sometimes)
		if i >= len(quote.Close) || i >= len(quote.Open) {
			continue
		}

		prices = append(prices, PriceData{
			Date:   time.Unix(result.Timestamp[i], 0),
			Open:   quote.Open[i],
			High:   quote.High[i],
			Low:    quote.Low[i],
			Close:  quote.Close[i],
			Volume: quote.Volume[i],
		})
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
