package schedule

import (
	"fmt"
	"time"

	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
)

// ForeignAssetsA3Record represents a record in Schedule FA A3
type ForeignAssetsA3Record struct {
	ShareRecord              stock.ShareRecord
	TransactionType          stock.TransactionType
	DateOfAcquiringInterest  time.Time
	InitialValueOfInvestment float64
	PeakValueOfInvestment    float64
	ClosingValue             float64
	TotalGrossAmountPaid     float64
	TotalProceedsFromSale    float64

	// Additional tracking fields
	PeakClosingDate     time.Time
	PeakClosingValue    float64
	YearEndClosingValue float64
	LastTradingDate     time.Time
}

// ForeignAssetsA3 represents Schedule FA A3
type ForeignAssetsA3 struct {
	Records       []ForeignAssetsA3Record
	FinancialYear stock.FinancialYear
}

// GenerateScheduleFAA3 generates Schedule FA A3
func GenerateScheduleFAA3(
	sharesIssued []stock.ShareIssuedRecord,
	sharesSold []stock.ShareSoldRecord,
	yahooClient *stock.YahooClient,
	forexRates *forex.SBIReferenceRates,
	financialYear stock.FinancialYear,
) (*ForeignAssetsA3, error) {
	schedule := &ForeignAssetsA3{
		FinancialYear: financialYear,
	}

	// Process shares issued (still held at year end)
	for _, share := range sharesIssued {
		// Skip if issued after year end
		if share.IssueDate.After(financialYear.End) {
			continue
		}

		record, err := createFARecordForIssuedShare(share, yahooClient, forexRates, financialYear.End)
		if err != nil {
			return nil, fmt.Errorf("processing issued share: %w", err)
		}

		schedule.Records = append(schedule.Records, record)
	}

	// Process shares sold
	for _, share := range sharesSold {
		// Skip if sold before year start or issued after year end
		if share.SaleDate.Before(financialYear.Start) || share.IssueDate.After(financialYear.End) {
			continue
		}

		record, err := createFARecordForSoldShare(share, yahooClient, forexRates, financialYear.End)
		if err != nil {
			return nil, fmt.Errorf("processing sold share: %w", err)
		}

		schedule.Records = append(schedule.Records, record)
	}

	return schedule, nil
}

func createFARecordForIssuedShare(
	share stock.ShareIssuedRecord,
	yahooClient *stock.YahooClient,
	forexRates *forex.SBIReferenceRates,
	yearEnd time.Time,
) (ForeignAssetsA3Record, error) {
	// Get exchange rate for issue date
	_, issueRate, err := forexRates.GetRate(share.IssueDate, true)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting issue date rate: %w", err)
	}

	// Get peak closing value between issue date and year end
	peakClose, peakDate, err := yahooClient.GetPeakClosingValue(share.Ticker, share.IssueDate, yearEnd)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting peak value: %w", err)
	}

	// Get exchange rate for peak date
	_, peakRate, err := forexRates.GetRate(peakDate, true)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting peak date rate: %w", err)
	}

	// Get year-end closing value
	yearEndClose, lastTradingDate, err := yahooClient.GetClosingPriceOnDate(share.Ticker, yearEnd)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting year-end value: %w", err)
	}

	// Get exchange rate for year end (don't adjust to previous month for year-end)
	_, yearEndRate, err := forexRates.GetRate(yearEnd, false)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting year-end rate: %w", err)
	}

	return ForeignAssetsA3Record{
		ShareRecord:              share,
		TransactionType:          stock.TransactionIssued,
		DateOfAcquiringInterest:  share.IssueDate,
		InitialValueOfInvestment: float64(share.SharesIssued) * share.FMVPerShare * issueRate.TTBuyExchangeRate,
		PeakValueOfInvestment:    float64(share.SharesIssued) * peakClose * peakRate.TTBuyExchangeRate,
		ClosingValue:             float64(share.SharesIssued) * yearEndClose * yearEndRate.TTBuyExchangeRate,
		TotalGrossAmountPaid:     0, // No dividends
		TotalProceedsFromSale:    0, // Not sold
		PeakClosingDate:          peakDate,
		PeakClosingValue:         peakClose,
		YearEndClosingValue:      yearEndClose,
		LastTradingDate:          lastTradingDate,
	}, nil
}

func createFARecordForSoldShare(
	share stock.ShareSoldRecord,
	yahooClient *stock.YahooClient,
	forexRates *forex.SBIReferenceRates,
	yearEnd time.Time,
) (ForeignAssetsA3Record, error) {
	// Get exchange rates
	_, issueRate, err := forexRates.GetRate(share.IssueDate, true)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting issue date rate: %w", err)
	}

	_, saleRate, err := forexRates.GetRate(share.SaleDate, true)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting sale date rate: %w", err)
	}

	// Get peak closing value between issue date and sale date
	peakClose, peakDate, err := yahooClient.GetPeakClosingValue(share.Ticker, share.IssueDate, share.SaleDate)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting peak value: %w", err)
	}

	_, peakRate, err := forexRates.GetRate(peakDate, true)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting peak date rate: %w", err)
	}

	return ForeignAssetsA3Record{
		ShareRecord:              share,
		TransactionType:          stock.TransactionSold,
		DateOfAcquiringInterest:  share.IssueDate,
		InitialValueOfInvestment: float64(share.SharesSold) * share.FMVOnIssueDate * issueRate.TTBuyExchangeRate,
		PeakValueOfInvestment:    float64(share.SharesSold) * peakClose * peakRate.TTBuyExchangeRate,
		ClosingValue:             0, // Sold before year end
		TotalGrossAmountPaid:     0, // No dividends
		TotalProceedsFromSale:    float64(share.SharesSold) * share.FMVOnSaleDate * saleRate.TTBuyExchangeRate,
		PeakClosingDate:          peakDate,
		PeakClosingValue:         peakClose,
	}, nil
}
