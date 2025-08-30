package stock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYahooClient_FetchHistoricalData(t *testing.T) {
	client := NewYahooClient()

	tests := []struct {
		name             string
		ticker           string
		startDate        time.Time
		endDate          time.Time
		expectedErrorStr string
	}{
		{
			name:      "Valid ticker and date range",
			ticker:    "AAPL",
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:             "Invalid ticker",
			ticker:           "INVALIDTICKER123",
			startDate:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:          time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			expectedErrorStr: "yahoo API error: No data found, symbol may be delisted",
		},
		{
			name:             "End date before start date",
			ticker:           "AAPL",
			startDate:        time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			endDate:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expectedErrorStr: "yahoo API error: Invalid input - start date cannot be after end date.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := client.FetchHistoricalData(tt.ticker, tt.startDate, tt.endDate)
			if err != nil {
				if tt.expectedErrorStr != "" {
					require.Contains(t, err.Error(), tt.expectedErrorStr)
					return
				}
				t.Errorf("FetchHistoricalData() error = %v, expectedErrorStr %v", err, tt.expectedErrorStr)
			}

			assert.NotNil(t, data)
			assert.Greater(t, len(data), 0, "Expected data but got none")
		})
	}
}

func TestYahooClient_GetPeakClosingValue(t *testing.T) {
	client := NewYahooClient()

	// Use a known date range where we can verify
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	maxClose, maxDate, err := client.GetPeakClosingValue("AAPL", startDate, endDate)
	if err != nil {
		t.Fatalf("GetPeakClosingValue() error = %v", err)
	}

	if maxClose <= 0 {
		t.Errorf("Expected positive max close, got %f", maxClose)
	}

	if maxDate.Before(startDate) || maxDate.After(endDate) {
		t.Errorf("Peak date %v outside range %v - %v", maxDate, startDate, endDate)
	}
}

func TestYahooClient_GetClosingPriceOnDate(t *testing.T) {
	client := NewYahooClient()

	// Test year-end closing (market might be closed on Dec 31)
	targetDate := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)

	price, actualDate, err := client.GetClosingPriceOnDate("AAPL", targetDate)
	if err != nil {
		t.Fatalf("GetClosingPriceOnDate() error = %v", err)
	}

	if price <= 0 {
		t.Errorf("Expected positive closing price, got %f", price)
	}

	// Actual date should be on or before target date
	if actualDate.After(targetDate) {
		t.Errorf("Actual date %v is after target date %v", actualDate, targetDate)
	}

	// But not too far before (max 10 days as per implementation)
	maxPastDate := targetDate.AddDate(0, 0, -10)
	if actualDate.Before(maxPastDate) {
		t.Errorf("Actual date %v is too far before target date %v", actualDate, targetDate)
	}
}

func TestYahooClient_Caching(t *testing.T) {
	client := NewYahooClient()

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	// First call - should hit API
	data1, err := client.FetchHistoricalData("AAPL", startDate, endDate)
	if err != nil {
		t.Fatalf("First fetch error = %v", err)
	}

	// Second call - should use cache
	data2, err := client.FetchHistoricalData("AAPL", startDate, endDate)
	if err != nil {
		t.Fatalf("Second fetch error = %v", err)
	}

	// Data should be identical
	if len(data1) != len(data2) {
		t.Errorf("Cached data length mismatch: %d vs %d", len(data1), len(data2))
	}
}
