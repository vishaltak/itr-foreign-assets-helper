package schedule

import (
	"fmt"
	"time"

	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
)

// ForeignAssetsA3Record represents a record in Schedule FA A3.
//
// Each ValuationDate carries the event date, the SBI TT-buy rate applied, and
// the date that rate was taken from. SaleDate is a zero ValuationDate for
// issued (still-held) records; YearEnd is zero for sold records.
type ForeignAssetsA3Record struct {
	ShareRecord     stock.ShareRecord
	TransactionType stock.TransactionType

	IssueDate ValuationDate // Date = date of acquiring interest
	SaleDate  ValuationDate
	PeakClose ValuationDate // Date = peak closing date
	YearEnd   ValuationDate // Date = last trading date on/before 31 Dec

	PeakClosingValue    float64
	YearEndClosingValue float64

	InitialValueOfInvestment float64
	PeakValueOfInvestment    float64
	ClosingValue             float64
	TotalGrossAmountPaid     float64
	TotalProceedsFromSale    float64
}

// ForeignAssetsA3 represents Schedule FA A3
type ForeignAssetsA3 struct {
	Records       []ForeignAssetsA3Record
	FinancialYear stock.FinancialYear
}

// PriceProvider supplies historical stock prices.
// *stock.YahooClient implements it; tests can supply a deterministic fake.
type PriceProvider interface {
	GetPeakClosingValue(ticker string, startDate, endDate time.Time) (float64, time.Time, error)
	GetClosingPriceOnDate(ticker string, targetDate time.Time) (float64, time.Time, error)
}

// GenerateScheduleFAA3 generates Schedule FA A3.
//
// Schedule FA is reported for the calendar year of the financial year's start
// (1 January to 31 December, both inclusive), unlike Schedule CG/AL which use
// the Indian financial year (1 April to 31 March).
func GenerateScheduleFAA3(
	sharesIssued []stock.ShareIssuedRecord,
	sharesSold []stock.ShareSoldRecord,
	priceProvider PriceProvider,
	forexRates *forex.SBIReferenceRates,
	financialYear stock.FinancialYear,
) (*ForeignAssetsA3, error) {
	schedule := &ForeignAssetsA3{
		FinancialYear: financialYear,
	}

	calendarYearStart := financialYear.ForeignAssetsStart()
	calendarYearEnd := financialYear.ForeignAssetsEnd()

	// Process shares issued (still held at year end)
	for _, share := range sharesIssued {
		// Skip if issued after the calendar year end (31 December).
		if share.IssueDate.After(calendarYearEnd) {
			continue
		}

		record, err := createFARecordForIssuedShare(share, priceProvider, forexRates, calendarYearEnd)
		if err != nil {
			return nil, fmt.Errorf("processing issued share: %w", err)
		}

		schedule.Records = append(schedule.Records, record)
	}

	// Process shares sold
	for _, share := range sharesSold {
		// Skip if sold before the calendar year start (1 January) or issued
		// after the calendar year end (31 December).
		if share.SaleDate.Before(calendarYearStart) || share.IssueDate.After(calendarYearEnd) {
			continue
		}

		record, err := createFARecordForSoldShare(share, priceProvider, forexRates)
		if err != nil {
			return nil, fmt.Errorf("processing sold share: %w", err)
		}

		schedule.Records = append(schedule.Records, record)
	}

	return schedule, nil
}

func createFARecordForIssuedShare(
	share stock.ShareIssuedRecord,
	priceProvider PriceProvider,
	forexRates *forex.SBIReferenceRates,
	yearEnd time.Time,
) (ForeignAssetsA3Record, error) {
	// Get exchange rate for issue date
	issueRate, err := forexRates.GetRate(share.IssueDate, true)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting issue date rate: %w", err)
	}

	// Get peak closing value between issue date and year end
	peakClose, peakDate, err := priceProvider.GetPeakClosingValue(share.Ticker, share.IssueDate, yearEnd)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting peak value: %w", err)
	}

	// Get exchange rate for peak date
	peakRate, err := forexRates.GetRate(peakDate, true)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting peak date rate: %w", err)
	}

	// Get year-end closing value
	yearEndClose, lastTradingDate, err := priceProvider.GetClosingPriceOnDate(share.Ticker, yearEnd)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting year-end value: %w", err)
	}

	// Get exchange rate for year end (don't adjust to previous month for year-end)
	yearEndRate, err := forexRates.GetRate(yearEnd, false)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting year-end rate: %w", err)
	}

	return ForeignAssetsA3Record{
		ShareRecord:     share,
		TransactionType: stock.TransactionIssued,
		IssueDate:       ValuationDate{Date: share.IssueDate, Rate: issueRate},
		PeakClose:       ValuationDate{Date: peakDate, Rate: peakRate},
		YearEnd:         ValuationDate{Date: lastTradingDate, Rate: yearEndRate},

		PeakClosingValue:    peakClose,
		YearEndClosingValue: yearEndClose,

		InitialValueOfInvestment: share.SharesIssued * share.FMVOnIssueDate * issueRate.TTBuyExchangeRate,
		PeakValueOfInvestment:    share.SharesIssued * peakClose * peakRate.TTBuyExchangeRate,
		ClosingValue:             share.SharesIssued * yearEndClose * yearEndRate.TTBuyExchangeRate,
		TotalGrossAmountPaid:     0, // No dividends
		TotalProceedsFromSale:    0, // Not sold
	}, nil
}

func createFARecordForSoldShare(
	share stock.ShareSoldRecord,
	priceProvider PriceProvider,
	forexRates *forex.SBIReferenceRates,
) (ForeignAssetsA3Record, error) {
	// Get exchange rates
	issueRate, err := forexRates.GetRate(share.IssueDate, true)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting issue date rate: %w", err)
	}

	saleRate, err := forexRates.GetRate(share.SaleDate, true)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting sale date rate: %w", err)
	}

	// Get peak closing value between issue date and sale date
	peakClose, peakDate, err := priceProvider.GetPeakClosingValue(share.Ticker, share.IssueDate, share.SaleDate)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting peak value: %w", err)
	}

	peakRate, err := forexRates.GetRate(peakDate, true)
	if err != nil {
		return ForeignAssetsA3Record{}, fmt.Errorf("getting peak date rate: %w", err)
	}

	return ForeignAssetsA3Record{
		ShareRecord:     share,
		TransactionType: stock.TransactionSold,
		IssueDate:       ValuationDate{Date: share.IssueDate, Rate: issueRate},
		SaleDate:        ValuationDate{Date: share.SaleDate, Rate: saleRate},
		PeakClose:       ValuationDate{Date: peakDate, Rate: peakRate},

		PeakClosingValue: peakClose,

		InitialValueOfInvestment: share.SharesSold * share.FMVOnIssueDate * issueRate.TTBuyExchangeRate,
		PeakValueOfInvestment:    share.SharesSold * peakClose * peakRate.TTBuyExchangeRate,
		ClosingValue:             0, // Sold before year end
		TotalGrossAmountPaid:     0, // No dividends
		TotalProceedsFromSale:    share.SharesSold * share.FMVOnSaleDate * saleRate.TTBuyExchangeRate,
	}, nil
}
