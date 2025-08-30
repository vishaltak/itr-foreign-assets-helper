package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
)

func createMockForexRates() *forex.SBIReferenceRates {
	rates := forex.NewSBIReferenceRates()

	// Add some test rates
	for i := 1; i <= 31; i++ {
		for month := 1; month <= 12; month++ {
			date := time.Date(2023, time.Month(month), i, 0, 0, 0, 0, time.UTC)
			rates.Rates[date.Format(time.DateOnly)] = forex.ReferenceRate{
				Date:              date,
				SourceCurrency:    "USD",
				TargetCurrency:    "INR",
				TTBuyExchangeRate: 82.50,
			}

			date = time.Date(2024, time.Month(month), i, 0, 0, 0, 0, time.UTC)
			rates.Rates[date.Format(time.DateOnly)] = forex.ReferenceRate{
				Date:              date,
				SourceCurrency:    "USD",
				TargetCurrency:    "INR",
				TTBuyExchangeRate: 83.00,
			}
		}
	}

	return rates
}

func TestGenerateScheduleAL(t *testing.T) {
	forexRates := createMockForexRates()

	sharesIssued := []stock.ShareIssuedRecord{
		{
			Ticker:       "AAPL",
			AwardNumber:  12345,
			SharesIssued: 100,
			IssueDate:    time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
			FMVPerShare:  150.00,
		},
	}

	financialYear, err := stock.ParseFinancialYear("2023-2024")
	require.NoError(t, err)

	schedule, err := GenerateScheduleAL(sharesIssued, forexRates, *financialYear)
	if err != nil {
		t.Fatalf("GenerateScheduleAL() error = %v", err)
	}

	if len(schedule.Records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(schedule.Records))
	}

	if len(schedule.Records) > 0 {
		record := schedule.Records[0]

		// Cost of acquisition = shares * FMV * exchange rate
		expectedCost := 100 * 150.00 * 82.50
		if record.CostOfAcquisition != expectedCost {
			t.Errorf("Expected cost %f, got %f", expectedCost, record.CostOfAcquisition)
		}
	}
}
