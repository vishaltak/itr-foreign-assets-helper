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

	// A distinct rate per year lets tests assert which date's rate was used.
	ratesByYear := map[int]float64{
		2023: 82.50,
		2024: 83.00,
		2025: 84.00,
		2026: 85.00,
	}

	for year, rate := range ratesByYear {
		for i := 1; i <= 31; i++ {
			for month := 1; month <= 12; month++ {
				date := time.Date(year, time.Month(month), i, 0, 0, 0, 0, time.UTC)
				rates.Rates[date.Format(time.DateOnly)] = forex.ReferenceRate{
					Date:              date,
					SourceCurrency:    "USD",
					TargetCurrency:    "INR",
					TTBuyExchangeRate: rate,
				}
			}
		}
	}

	return rates
}

func TestGenerateScheduleAL_HeldAsOnFinancialYearEnd(t *testing.T) {
	forexRates := createMockForexRates()

	// FY 2025-2026 => Schedule AL reflects holdings as on 31 Mar 2026.
	heldAtYearEnd := stock.ShareIssuedRecord{
		Ticker:       "AAPL",
		AwardNumber:  "1",
		SharesIssued: 10,
		IssueDate:    time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC), // Jan-Mar 2026
		FMVPerShare:  100.00,
	}
	// Issued after the FY end - belongs to the next FY, must be excluded.
	issuedNextFY := stock.ShareIssuedRecord{
		Ticker:       "MSFT",
		AwardNumber:  "2",
		SharesIssued: 10,
		IssueDate:    time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		FMVPerShare:  100.00,
	}

	financialYear, err := stock.ParseFinancialYear("2025-2026")
	require.NoError(t, err)

	schedule, err := GenerateScheduleAL([]stock.ShareIssuedRecord{heldAtYearEnd, issuedNextFY}, forexRates, *financialYear)
	require.NoError(t, err)

	require.Len(t, schedule.Records, 1)
	require.Equal(t, "AAPL", schedule.Records[0].ShareRecord.Ticker)
	// Issue rate uses last day of previous month (Jan 2026 => 85.00).
	require.Equal(t, 10*100.00*85.00, schedule.Records[0].CostOfAcquisition)
}

func TestGenerateScheduleAL(t *testing.T) {
	forexRates := createMockForexRates()

	sharesIssued := []stock.ShareIssuedRecord{
		{
			Ticker:       "AAPL",
			AwardNumber:  "12345",
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
