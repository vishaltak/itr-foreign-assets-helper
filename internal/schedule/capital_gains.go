package schedule

import (
	"fmt"
	"math"

	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
)

// ProceedsRoundingDiscrepancy reports, across the given sold lots, the
// aggregate and largest single-lot INR difference between two ways of valuing
// sale proceeds: quantity x (rounded) per-share price versus ETrade's reported
// Total Proceeds. The schedules use Total Proceeds; this quantifies how much
// the per-share reconstruction would have differed, so the run can advise the
// user. Each lot is converted at its sale-date TT-buy rate; lots without a rate
// are skipped (they are not reported anyway).
func ProceedsRoundingDiscrepancy(sharesSold []stock.ShareSoldRecord, forexRates *forex.SBIReferenceRates) (total, largest float64) {
	for _, s := range sharesSold {
		rate, err := forexRates.GetRate(s.SaleDate, true)
		if err != nil {
			continue
		}
		diff := math.Abs(s.SharesSold*s.FMVOnSaleDate-s.TotalProceeds) * rate.TTBuyExchangeRate
		total += diff
		if diff > largest {
			largest = diff
		}
	}
	return total, largest
}

// CapitalGainsRecord represents a capital gains record
type CapitalGainsRecord struct {
	ShareRecord              stock.ShareSoldRecord
	IssueDate                ValuationDate
	SaleDate                 ValuationDate
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
		issueRate, err := forexRates.GetRate(share.IssueDate, true)
		if err != nil {
			return nil, fmt.Errorf("getting issue rate: %w", err)
		}

		saleRate, err := forexRates.GetRate(share.SaleDate, true)
		if err != nil {
			return nil, fmt.Errorf("getting sale rate: %w", err)
		}

		costOfAcquisition := share.SharesSold * share.FMVOnIssueDate * issueRate.TTBuyExchangeRate
		// Use ETrade's reported Total Proceeds (the actual amount received), not
		// quantity x rounded per-share price.
		fullValue := share.TotalProceeds * saleRate.TTBuyExchangeRate

		record := CapitalGainsRecord{
			ShareRecord:              share,
			IssueDate:                ValuationDate{Date: share.IssueDate, Rate: issueRate},
			SaleDate:                 ValuationDate{Date: share.SaleDate, Rate: saleRate},
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
