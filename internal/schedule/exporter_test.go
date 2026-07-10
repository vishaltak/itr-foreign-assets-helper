package schedule

import (
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
	"github.com/xuri/excelize/v2"
)

func TestExportToExcel_RoundTrip(t *testing.T) {
	issueDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	issueRateDate := time.Date(2025, 5, 31, 0, 0, 0, 0, time.UTC)
	saleDate := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	saleRateDate := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	yearEndDate := time.Date(2025, 12, 29, 0, 0, 0, 0, time.UTC)
	yearEndRateDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	faSchedule := &ForeignAssetsA3{
		Records: []ForeignAssetsA3Record{
			{
				ShareRecord: stock.ShareIssuedRecord{
					SourceMetadata: stock.SourceMetadata{FileName: "/tmp/holdings.xlsx", SheetName: "Sellable", Row: 2},
					Broker:         "ETrade",
					Ticker:         "AAPL",
					AwardNumber:    "A-1",
					SharesIssued:   10,
					IssueDate:      issueDate,
					FMVOnIssueDate: 100,
				},
				TransactionType:          stock.TransactionIssued,
				IssueDate:                ValuationDate{Date: issueDate, Rate: forex.ReferenceRate{Date: issueRateDate, TTBuyExchangeRate: 83.0}},
				PeakClose:                ValuationDate{Date: issueDate, Rate: forex.ReferenceRate{Date: issueRateDate, TTBuyExchangeRate: 83.0}},
				YearEnd:                  ValuationDate{Date: yearEndDate, Rate: forex.ReferenceRate{Date: yearEndRateDate, TTBuyExchangeRate: 83.0}},
				PeakClosingValue:         250,
				YearEndClosingValue:      300,
				InitialValueOfInvestment: 84000,
				PeakValueOfInvestment:    210000,
				ClosingValue:             252000,
			},
		},
	}

	cgSchedule := &CapitalGains{
		Records: []CapitalGainsRecord{
			{
				ShareRecord: stock.ShareSoldRecord{
					SourceMetadata: stock.SourceMetadata{FileName: "/tmp/gains.xlsx", SheetName: "G&L_Expanded", Row: 3},
					Broker:         "ETrade",
					Ticker:         "MSFT",
					AwardNumber:    "A-2",
					IssueDate:      issueDate,
					FMVOnIssueDate: 100,
					SharesSold:     10,
					SaleDate:       saleDate,
					FMVOnSaleDate:  200,
				},
				IssueDate:                ValuationDate{Date: issueDate, Rate: forex.ReferenceRate{Date: issueRateDate, TTBuyExchangeRate: 83.0}},
				SaleDate:                 ValuationDate{Date: saleDate, Rate: forex.ReferenceRate{Date: saleRateDate, TTBuyExchangeRate: 85.0}},
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
					SourceMetadata: stock.SourceMetadata{FileName: "/tmp/holdings.xlsx", SheetName: "Sellable", Row: 2},
					Broker:         "ETrade",
					Ticker:         "GOOG",
					AwardNumber:    "A-3",
					SharesIssued:   5,
					IssueDate:      issueDate,
					FMVOnIssueDate: 100,
				},
				IssueDate:         ValuationDate{Date: issueDate, Rate: forex.ReferenceRate{Date: issueRateDate, TTBuyExchangeRate: 83.0}},
				CostOfAcquisition: 42000,
			},
		},
	}

	outFile := filepath.Join(t.TempDir(), "out.xlsx")
	require.NoError(t, ExportToExcel(faSchedule, cgSchedule, alSchedule, outFile))

	f, err := excelize.OpenFile(outFile)
	require.NoError(t, err)
	defer f.Close()

	require.ElementsMatch(t, []string{"Schedule_FA_A3", "Schedule_CG", "Schedule_AL"}, f.GetSheetList())

	// headerCols maps each header name to its column letter for a sheet.
	headerCols := func(sheet string) map[string]string {
		rows, rErr := f.GetRows(sheet)
		require.NoError(t, rErr)
		m := map[string]string{}
		for i, h := range rows[0] {
			col, _ := excelize.ColumnNumberToName(i + 1)
			m[h] = col
		}
		return m
	}
	// cell reads a cell by header name and row number (formatted value).
	cell := func(sheet string, cols map[string]string, header string, row int) string {
		col, ok := cols[header]
		require.Truef(t, ok, "sheet %q missing column %q", sheet, header)
		v, cErr := f.GetCellValue(sheet, col+strconv.Itoa(row))
		require.NoError(t, cErr)
		return v
	}

	// --- Schedule FA A3 ---
	fa := headerCols("Schedule_FA_A3")
	require.Equal(t, "holdings.xlsx", cell("Schedule_FA_A3", fa, "Source File", 2))
	require.Equal(t, "Sellable", cell("Schedule_FA_A3", fa, "Sheet", 2))
	require.Equal(t, "issued", cell("Schedule_FA_A3", fa, "Transaction Type", 2))
	require.Equal(t, "AAPL", cell("Schedule_FA_A3", fa, "Ticker", 2))
	require.Equal(t, "10", cell("Schedule_FA_A3", fa, "Shares", 2)) // unformatted, as-is
	require.Equal(t, "2025-06-01", cell("Schedule_FA_A3", fa, "Issue Date", 2))
	require.Equal(t, "$100.00", cell("Schedule_FA_A3", fa, "FMV on Issue Date", 2))
	require.Equal(t, "2025-05-31", cell("Schedule_FA_A3", fa, "TT Buy Rate Date Considered for Issue Date", 2))
	require.Equal(t, "₹83.00", cell("Schedule_FA_A3", fa, "TT Buy Rate Considered for Issue Date", 2))
	require.Equal(t, "", cell("Schedule_FA_A3", fa, "Sale Date", 2)) // issued -> blank
	require.Equal(t, "2025-12-29", cell("Schedule_FA_A3", fa, "Year End Date", 2))
	require.Equal(t, "₹252,000.00", cell("Schedule_FA_A3", fa, "Closing Value", 2))

	// --- Schedule CG ---
	cg := headerCols("Schedule_CG")
	require.Equal(t, "sold", cell("Schedule_CG", cg, "Transaction Type", 2))
	require.Equal(t, "MSFT", cell("Schedule_CG", cg, "Ticker", 2))
	require.Equal(t, "2026-02-10", cell("Schedule_CG", cg, "Sale Date", 2))
	require.Equal(t, "₹85.00", cell("Schedule_CG", cg, "TT Buy Rate Considered for Sale Date", 2))
	require.Equal(t, "₹86,000.00", cell("Schedule_CG", cg, "Short Term Capital Gain", 2))

	// --- Schedule AL ---
	al := headerCols("Schedule_AL")
	require.Equal(t, "issued", cell("Schedule_AL", al, "Transaction Type", 2))
	require.Equal(t, "GOOG", cell("Schedule_AL", al, "Ticker", 2))
	require.Equal(t, "₹42,000.00", cell("Schedule_AL", al, "Cost of Acquisition without indexation", 2))
	_, hasImprovement := al["Cost of Improvement without indexation"]
	require.False(t, hasImprovement, "Schedule AL must not have a Cost of Improvement column")
}
