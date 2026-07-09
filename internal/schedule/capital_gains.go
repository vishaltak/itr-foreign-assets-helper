package schedule

import (
	"fmt"

	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
)

// CapitalGainsRecord represents a capital gains record
type CapitalGainsRecord struct {
	ShareRecord              stock.ShareSoldRecord
	CostOfAcquisition        float64
	CostOfImprovement        float64
	ExpenditureOnTransfer    float64
	FullValueOfConsideration float64
	ShortTermCapitalGain     float64
}

// CapitalGains represents Schedule CG
type CapitalGains struct {
	Records       []CapitalGainsRecord
	FinancialYear stock.FinancialYear
}

// GenerateScheduleCG generates Schedule CG for capital gains
func GenerateScheduleCG(
	sharesSold []stock.ShareSoldRecord,
	forexRates *forex.SBIReferenceRates,
	financialYear stock.FinancialYear,
) (*CapitalGains, error) {
	schedule := &CapitalGains{
		FinancialYear: financialYear,
	}

	for _, share := range sharesSold {
		// Include only sales within the financial year
		if share.SaleDate.Before(financialYear.Start) || share.SaleDate.After(financialYear.End) {
			continue
		}

		// Skip if issued after financial year end
		if share.IssueDate.After(financialYear.End) {
			continue
		}

		// Get exchange rates
		_, issueRate, err := forexRates.GetRate(share.IssueDate, true)
		if err != nil {
			return nil, fmt.Errorf("getting issue rate: %w", err)
		}

		_, saleRate, err := forexRates.GetRate(share.SaleDate, true)
		if err != nil {
			return nil, fmt.Errorf("getting sale rate: %w", err)
		}

		costOfAcquisition := share.SharesSold * share.FMVOnIssueDate * issueRate.TTBuyExchangeRate
		fullValue := share.SharesSold * share.FMVOnSaleDate * saleRate.TTBuyExchangeRate

		record := CapitalGainsRecord{
			ShareRecord:              share,
			CostOfAcquisition:        costOfAcquisition,
			CostOfImprovement:        0,
			ExpenditureOnTransfer:    0,
			FullValueOfConsideration: fullValue,
			ShortTermCapitalGain:     fullValue - costOfAcquisition,
		}

		schedule.Records = append(schedule.Records, record)
	}

	return schedule, nil
}
