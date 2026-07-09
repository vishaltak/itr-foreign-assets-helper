package schedule

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
	"github.com/xuri/excelize/v2"
)

func TestExportToExcel_RoundTrip(t *testing.T) {
	issueDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	saleDate := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	faSchedule := &ForeignAssetsA3{
		Records: []ForeignAssetsA3Record{
			{
				ShareRecord: stock.ShareIssuedRecord{
					SourceMetadata: stock.SourceMetadata{FileName: "/tmp/holdings.xlsx", Row: 2},
					Broker:         "ETrade",
					Ticker:         "AAPL",
					AwardNumber:    "A-1",
					SharesIssued:   10,
					IssueDate:      issueDate,
					FMVPerShare:    100,
				},
				TransactionType:          stock.TransactionIssued,
				DateOfAcquiringInterest:  issueDate,
				InitialValueOfInvestment: 84000,
				PeakValueOfInvestment:    210000,
				ClosingValue:             252000,
				PeakClosingDate:          issueDate,
				PeakClosingValue:         250,
				YearEndClosingValue:      300,
				LastTradingDate:          issueDate,
			},
		},
	}

	cgSchedule := &CapitalGains{
		Records: []CapitalGainsRecord{
			{
				ShareRecord: stock.ShareSoldRecord{
					SourceMetadata: stock.SourceMetadata{FileName: "/tmp/gains.xlsx", Row: 3},
					Broker:         "ETrade",
					Ticker:         "MSFT",
					AwardNumber:    "A-2",
					IssueDate:      issueDate,
					FMVOnIssueDate: 100,
					SharesSold:     10,
					SaleDate:       saleDate,
					FMVOnSaleDate:  200,
				},
				CostOfAcquisition:        84000,
				FullValueOfConsideration: 170000,
				ShortTermCapitalGain:     86000,
			},
		},
	}

	alSchedule := &AssetsAndLiabilities{
		Records: []AssetsAndLiabilitiesRecord{
			{
				ShareRecord: stock.ShareIssuedRecord{
					SourceMetadata: stock.SourceMetadata{FileName: "/tmp/holdings.xlsx", Row: 2},
					Broker:         "ETrade",
					Ticker:         "GOOG",
					AwardNumber:    "A-3",
					SharesIssued:   5,
					IssueDate:      issueDate,
					FMVPerShare:    100,
				},
				CostOfAcquisition: 42000,
			},
		},
	}

	outFile := filepath.Join(t.TempDir(), "out.xlsx")
	require.NoError(t, ExportToExcel(faSchedule, cgSchedule, alSchedule, outFile))

	f, err := excelize.OpenFile(outFile)
	require.NoError(t, err)
	defer f.Close()

	sheets := f.GetSheetList()
	require.ElementsMatch(t, []string{"Schedule_FA_A3", "Schedule_CG", "Schedule_AL"}, sheets)

	cell := func(sheet, ref string) string {
		v, cellErr := f.GetCellValue(sheet, ref)
		require.NoError(t, cellErr)
		return v
	}

	// Schedule FA A3: headers + representative data.
	require.Equal(t, "Source File", cell("Schedule_FA_A3", "A1"))
	require.Equal(t, "holdings.xlsx", cell("Schedule_FA_A3", "A2")) // base name only
	require.Equal(t, "issued", cell("Schedule_FA_A3", "D2"))
	require.Equal(t, "AAPL", cell("Schedule_FA_A3", "F2"))
	require.Equal(t, "252000", cell("Schedule_FA_A3", "S2")) // Closing Value (INR)

	// Schedule CG: short-term capital gain lands in the last column.
	require.Equal(t, "MSFT", cell("Schedule_CG", "E2"))
	require.Equal(t, "86000", cell("Schedule_CG", "O2"))

	// Schedule AL: cost of acquisition.
	require.Equal(t, "GOOG", cell("Schedule_AL", "E2"))
	require.Equal(t, "42000", cell("Schedule_AL", "I2"))
}
