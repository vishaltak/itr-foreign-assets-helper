package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
)

func TestGenerateScheduleCG_FinancialYearWindow(t *testing.T) {
	forexRates := createMockForexRates()

	// FY 2025-2026 => Schedule CG covers sales from 1 Apr 2025 to 31 Mar 2026.
	insideFY := stock.ShareSoldRecord{
		Ticker:         "AAPL",
		AwardNumber:    "1",
		IssueDate:      time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
		FMVOnIssueDate: 100.00,
		SharesSold:     10,
		SaleDate:       time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC), // Jan-Mar 2026, inside FY
		FMVOnSaleDate:  200.00,
	}
	// Sold before the FY start (belongs to the previous FY) - must be excluded.
	beforeFY := stock.ShareSoldRecord{
		Ticker:         "MSFT",
		AwardNumber:    "2",
		IssueDate:      time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
		FMVOnIssueDate: 100.00,
		SharesSold:     10,
		SaleDate:       time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC),
		FMVOnSaleDate:  200.00,
	}
	// Sold after the FY end (belongs to the next FY) - must be excluded.
	afterFY := stock.ShareSoldRecord{
		Ticker:         "GOOG",
		AwardNumber:    "3",
		IssueDate:      time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
		FMVOnIssueDate: 100.00,
		SharesSold:     10,
		SaleDate:       time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), // after 31 Mar 2026
		FMVOnSaleDate:  200.00,
	}

	financialYear, err := stock.ParseFinancialYear("2025-2026")
	require.NoError(t, err)

	schedule, err := GenerateScheduleCG([]stock.ShareSoldRecord{insideFY, beforeFY, afterFY}, forexRates, *financialYear)
	require.NoError(t, err)

	require.Len(t, schedule.Records, 1)
	require.Equal(t, "AAPL", schedule.Records[0].ShareRecord.Ticker)
	require.Equal(t, insideFY.SaleDate, schedule.Records[0].ShareRecord.SaleDate)

	// Issue rate uses last day of previous month (Apr 2025 => 84.00),
	// sale rate uses last day of previous month (Jan 2026 => 85.00).
	require.Equal(t, 10*100.00*84.00, schedule.Records[0].CostOfAcquisition)
	require.Equal(t, 10*200.00*85.00, schedule.Records[0].FullValueOfConsideration)

	// The applied rates and the (adjusted) dates they came from are recorded.
	require.Equal(t, 84.00, schedule.Records[0].IssueDate.Rate.TTBuyExchangeRate)
	require.Equal(t, time.Date(2025, 4, 30, 0, 0, 0, 0, time.UTC), schedule.Records[0].IssueDate.Rate.Date)
	require.Equal(t, 85.00, schedule.Records[0].SaleDate.Rate.TTBuyExchangeRate)
	require.Equal(t, time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), schedule.Records[0].SaleDate.Rate.Date)
}

func TestGenerateScheduleCG(t *testing.T) {
	forexRates := createMockForexRates()

	sharesSold := []stock.ShareSoldRecord{
		{
			Ticker:         "AAPL",
			AwardNumber:    "12345",
			IssueDate:      time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
			FMVOnIssueDate: 140.00,
			SharesSold:     50,
			SaleDate:       time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC),
			FMVOnSaleDate:  180.00,
		},
	}

	financialYear, err := stock.ParseFinancialYear("2023-2024")
	require.NoError(t, err)

	schedule, err := GenerateScheduleCG(sharesSold, forexRates, *financialYear)
	if err != nil {
		t.Fatalf("GenerateScheduleCG() error = %v", err)
	}

	if len(schedule.Records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(schedule.Records))
	}

	if len(schedule.Records) > 0 {
		record := schedule.Records[0]

		// Cost of acquisition = shares * issue FMV * exchange rate
		expectedCost := 50 * 140.00 * 82.50
		if record.CostOfAcquisition != expectedCost {
			t.Errorf("Expected cost %f, got %f", expectedCost, record.CostOfAcquisition)
		}

		// Full value = shares * sale FMV * exchange rate
		expectedValue := 50 * 180.00 * 82.50
		if record.FullValueOfConsideration != expectedValue {
			t.Errorf("Expected value %f, got %f", expectedValue, record.FullValueOfConsideration)
		}

		// Capital gain = value - cost
		expectedGain := expectedValue - expectedCost
		if record.ShortTermCapitalGain != expectedGain {
			t.Errorf("Expected gain %f, got %f", expectedGain, record.ShortTermCapitalGain)
		}
	}
}
