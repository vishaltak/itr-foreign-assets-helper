package forex

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSBIReferenceRates_ParseCSV(t *testing.T) {
	csvData := `Date,TT Sell,TT Buy,Bill Sell,Bill Buy
2024-03-31 00:00,83.50,82.50,84.00,82.00
2024-03-30 00:00,83.45,82.45,83.95,81.95
2024-03-29 00:00,0.00,0.00,0.00,0.00
2024-02-29 00:00,83.00,82.00,83.50,81.50`

	rates := NewSBIReferenceRates()
	err := rates.parseCSV(strings.NewReader(csvData))
	if err != nil {
		t.Fatalf("parseCSV() error = %v", err)
	}

	// Check that zero Rates are filtered out
	d1, err := time.Parse("2006-01-02 15:04", "2024-03-29 09:01")
	if err != nil {
		t.Fatalf("time.Parse error = %v", err)
	}
	if _, ok := rates.Rates[d1.Format(time.DateOnly)]; ok {
		t.Error("Zero rate should have been filtered out")
	}

	// Check valid rate
	d2, err := time.Parse("2006-01-02 15:04", "2024-03-31 00:00")
	if err != nil {
		t.Fatalf("time.Parse error = %v", err)
	}
	rate, ok := rates.Rates[d2.Format(time.DateOnly)]
	if !ok {
		t.Error("Expected rate for 2024-03-31")
	}

	if rate.TTBuyExchangeRate != 82.50 {
		t.Errorf("Expected TT Buy rate 82.50, got %f", rate.TTBuyExchangeRate)
	}
}

func TestSBIReferenceRates_GetRate(t *testing.T) {
	rates := NewSBIReferenceRates()

	// Manually add some test Rates
	d1, err := time.Parse("2006-01-02 15:04", "2024-02-29 00:00")
	if err != nil {
		t.Fatalf("time.Parse error = %v", err)
	}
	rates.Rates[d1.Format(time.DateOnly)] = ReferenceRate{
		Date:              d1,
		SourceCurrency:    "USD",
		TargetCurrency:    "INR",
		TTBuyExchangeRate: 82.00,
	}
	d2, err := time.Parse("2006-01-02 15:04", "2024-03-31 00:00")
	if err != nil {
		t.Fatalf("time.Parse error = %v", err)
	}
	rates.Rates[d2.Format(time.DateOnly)] = ReferenceRate{
		Date:              d2,
		SourceCurrency:    "USD",
		TargetCurrency:    "INR",
		TTBuyExchangeRate: 83.00,
	}

	tests := []struct {
		name                      string
		date                      time.Time
		adjustToPreviousMonth     bool
		expectedDate              time.Time
		expectedTTBuyExchangeRate float64
		expectedErrorStr          string
	}{
		{
			name:                      "Exact date match",
			date:                      time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
			adjustToPreviousMonth:     false,
			expectedDate:              time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC),
			expectedTTBuyExchangeRate: 83.00,
		},
		{
			name:                      "Adjust to previous month",
			date:                      time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
			adjustToPreviousMonth:     true,
			expectedDate:              time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			expectedTTBuyExchangeRate: 82.00,
		},
		{
			name:                      "Fallback to earlier date",
			date:                      time.Date(2024, 3, 02, 0, 0, 0, 0, time.UTC),
			adjustToPreviousMonth:     false,
			expectedDate:              time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			expectedTTBuyExchangeRate: 82.00,
		},
		{
			name:                  "Gap larger than the lookback window fails loud",
			date:                  time.Date(2024, 3, 30, 0, 0, 0, 0, time.UTC),
			adjustToPreviousMonth: false,
			expectedErrorStr:      "no exchange rate found within 15 days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate, err := rates.GetRate(tt.date, tt.adjustToPreviousMonth)
			if err != nil {
				if tt.expectedErrorStr != "" {
					require.Contains(t, err.Error(), tt.expectedErrorStr)
					return
				}
				t.Errorf("GetRate() error = %v, expectedErrorStr %v", err, tt.expectedErrorStr)
			}

			assert.Equal(t, tt.expectedDate, rate.Date)
			assert.Equal(t, tt.expectedTTBuyExchangeRate, rate.TTBuyExchangeRate)
		})
	}
}
