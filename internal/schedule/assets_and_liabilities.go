package schedule

import (
	"fmt"

	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
)

// AssetsAndLiabilitiesRecord represents an asset/liability record
type AssetsAndLiabilitiesRecord struct {
	ShareRecord       stock.ShareIssuedRecord
	CostOfAcquisition float64
	CostOfImprovement float64
}

// AssetsAndLiabilities represents Schedule AL
type AssetsAndLiabilities struct {
	Records       []AssetsAndLiabilitiesRecord
	FinancialYear stock.FinancialYear
}

// GenerateScheduleAL generates Schedule AL for assets and liabilities
func GenerateScheduleAL(
	sharesIssued []stock.ShareIssuedRecord,
	forexRates *forex.SBIReferenceRates,
	financialYear stock.FinancialYear,
) (*AssetsAndLiabilities, error) {
	schedule := &AssetsAndLiabilities{
		FinancialYear: financialYear,
	}

	for _, share := range sharesIssued {
		// Include all shares held at financial year end
		if share.IssueDate.After(financialYear.End) {
			continue
		}

		// Get exchange rate
		_, issueRate, err := forexRates.GetRate(share.IssueDate, true)
		if err != nil {
			return nil, fmt.Errorf("getting issue rate: %w", err)
		}

		record := AssetsAndLiabilitiesRecord{
			ShareRecord:       share,
			CostOfAcquisition: share.SharesIssued * share.FMVPerShare * issueRate.TTBuyExchangeRate,
			CostOfImprovement: 0,
		}

		schedule.Records = append(schedule.Records, record)
	}

	return schedule, nil
}
