package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
)

// fakePriceProvider is a deterministic stand-in for the Yahoo Finance client
// so Schedule FA logic can be tested without network access.
type fakePriceProvider struct {
	peakClose    float64
	peakDate     time.Time
	yearEndClose float64
	yearEndDate  time.Time

	// captured for assertions
	lastPeakEnd       time.Time
	lastClosingTarget time.Time
}

func (f *fakePriceProvider) GetPeakClosingValue(ticker string, startDate, endDate time.Time) (float64, time.Time, error) {
	f.lastPeakEnd = endDate
	return f.peakClose, f.peakDate, nil
}

func (f *fakePriceProvider) GetClosingPriceOnDate(ticker string, targetDate time.Time) (float64, time.Time, error) {
	f.lastClosingTarget = targetDate
	return f.yearEndClose, f.yearEndDate, nil
}

func TestGenerateScheduleFAA3_CalendarYearFiltering(t *testing.T) {
	forexRates := createMockForexRates()
	prices := &fakePriceProvider{
		peakClose:    250.00,
		peakDate:     time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		yearEndClose: 300.00,
		yearEndDate:  time.Date(2025, 12, 29, 0, 0, 0, 0, time.UTC),
	}

	// FY 2025-2026 => Schedule FA covers the calendar year 1 Jan 2025 - 31 Dec 2025.
	sharesIssued := []stock.ShareIssuedRecord{
		{Ticker: "IN25", AwardNumber: "1", SharesIssued: 10, IssueDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), FMVOnIssueDate: 100},
		// Issued after 31 Dec 2025 - outside the FA calendar year, must be excluded.
		{Ticker: "OUT26", AwardNumber: "2", SharesIssued: 10, IssueDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), FMVOnIssueDate: 100},
	}
	sharesSold := []stock.ShareSoldRecord{
		// Sold Jan-Mar 2025 (inside the FA calendar year) - must be included.
		{Ticker: "SOLD25", AwardNumber: "3", IssueDate: time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC), FMVOnIssueDate: 100, SharesSold: 5, SaleDate: time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC), FMVOnSaleDate: 150},
		// Sold before 1 Jan 2025 - outside the FA calendar year, must be excluded.
		{Ticker: "SOLD24", AwardNumber: "4", IssueDate: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), FMVOnIssueDate: 100, SharesSold: 5, SaleDate: time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC), FMVOnSaleDate: 150},
	}

	financialYear, err := stock.ParseFinancialYear("2025-2026")
	require.NoError(t, err)

	schedule, err := GenerateScheduleFAA3(sharesIssued, sharesSold, prices, forexRates, *financialYear)
	require.NoError(t, err)

	tickers := map[string]bool{}
	for _, r := range schedule.Records {
		tickers[r.ShareRecord.GetTicker()] = true
	}
	require.True(t, tickers["IN25"], "share issued within calendar year should be included")
	require.True(t, tickers["SOLD25"], "share sold within calendar year (Jan-Mar) should be included")
	require.False(t, tickers["OUT26"], "share issued after 31 Dec should be excluded")
	require.False(t, tickers["SOLD24"], "share sold before 1 Jan should be excluded")
	require.Len(t, schedule.Records, 2)
}

func TestGenerateScheduleFAA3_UsesDecember31ForYearEnd(t *testing.T) {
	forexRates := createMockForexRates()
	prices := &fakePriceProvider{
		peakClose:    250.00,
		peakDate:     time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
		yearEndClose: 300.00,
		yearEndDate:  time.Date(2025, 12, 29, 0, 0, 0, 0, time.UTC),
	}

	sharesIssued := []stock.ShareIssuedRecord{
		{Ticker: "AAPL", AwardNumber: "1", SharesIssued: 10, IssueDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), FMVOnIssueDate: 100},
	}

	financialYear, err := stock.ParseFinancialYear("2025-2026")
	require.NoError(t, err)

	schedule, err := GenerateScheduleFAA3(sharesIssued, nil, prices, forexRates, *financialYear)
	require.NoError(t, err)
	require.Len(t, schedule.Records, 1)

	// Peak window end and year-end closing lookups must use 31 Dec 2025,
	// not the financial year end (31 Mar 2026).
	yearEnd := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	require.Equal(t, yearEnd, prices.lastPeakEnd)
	require.Equal(t, yearEnd, prices.lastClosingTarget)

	// Closing value uses the year-end (31 Dec 2025) rate, which is not
	// adjusted to the previous month => 2025 rate 84.00.
	require.Equal(t, 10*300.00*84.00, schedule.Records[0].ClosingValue)
}
