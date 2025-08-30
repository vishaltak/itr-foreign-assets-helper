package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
)

func TestGenerateScheduleCG(t *testing.T) {
	forexRates := createMockForexRates()

	sharesSold := []stock.ShareSoldRecord{
		{
			Ticker:         "AAPL",
			AwardNumber:    12345,
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
