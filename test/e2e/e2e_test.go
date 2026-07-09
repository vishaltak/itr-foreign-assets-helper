package e2e

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vtak/itr-foreign-assets-helper/internal/etrade"
	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
	"github.com/vtak/itr-foreign-assets-helper/internal/schedule"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
	"github.com/xuri/excelize/v2"
)

// TestEndToEnd runs the full pipeline against anonymized ETrade fixtures using
// the real SBI reference-rate data and live Yahoo Finance prices, then compares
// the generated workbook cell-by-cell against a committed golden workbook.
//
// It is opt-in (network-dependent) so the default suite stays hermetic:
//
//	RUN_E2E=1 go test ./test/e2e/
//
// When GTLB's historical prices legitimately change, refresh the golden with:
//
//	RUN_E2E=1 UPDATE_GOLDEN=1 go test ./test/e2e/
func TestEndToEnd(t *testing.T) {
	if os.Getenv("RUN_E2E") == "" {
		t.Skip("set RUN_E2E=1 to run the network end-to-end test")
	}

	const financialYearStr = "2025-2026"
	holdings := filepath.Join("testdata", "holdings.xlsx")
	gains := filepath.Join("testdata", "gains.xlsx")
	sbi := filepath.Join("testdata", "SBI_REFERENCE_RATES_USD.csv")
	goldenPath := filepath.Join("testdata", "expected.xlsx")

	forexRates := forex.NewSBIReferenceRates()
	require.NoError(t, forexRates.LoadFromFile(sbi))

	yahoo := stock.NewYahooClient()
	processor := etrade.NewProcessor(yahoo, forexRates)

	sharesIssued, err := processor.ProcessHoldings(holdings)
	require.NoError(t, err)
	require.NotEmpty(t, sharesIssued)

	sharesSold, err := processor.ProcessGainsAndLosses(gains)
	require.NoError(t, err)
	require.NotEmpty(t, sharesSold)

	fy, err := stock.ParseFinancialYear(financialYearStr)
	require.NoError(t, err)

	fa, err := schedule.GenerateScheduleFAA3(sharesIssued, sharesSold, yahoo, forexRates, *fy)
	require.NoError(t, err)
	require.NotEmpty(t, fa.Records)

	cg, err := schedule.GenerateScheduleCG(sharesSold, forexRates, *fy)
	require.NoError(t, err)

	al, err := schedule.GenerateScheduleAL(sharesIssued, forexRates, *fy)
	require.NoError(t, err)
	require.NotEmpty(t, al.Records)

	out := filepath.Join(t.TempDir(), "itr.xlsx")
	require.NoError(t, schedule.ExportToExcel(fa, cg, al, out))

	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, copyFile(out, goldenPath))
		t.Logf("updated golden workbook: %s", goldenPath)
		return
	}

	assertWorkbooksEqual(t, goldenPath, out)
}

// assertWorkbooksEqual compares two workbooks sheet-by-sheet and cell-by-cell.
func assertWorkbooksEqual(t *testing.T, goldenPath, actualPath string) {
	t.Helper()

	golden, err := excelize.OpenFile(goldenPath)
	require.NoError(t, err, "opening golden workbook (regenerate with UPDATE_GOLDEN=1)")
	defer golden.Close()

	actual, err := excelize.OpenFile(actualPath)
	require.NoError(t, err)
	defer actual.Close()

	require.ElementsMatch(t, golden.GetSheetList(), actual.GetSheetList(), "sheet names differ")

	for _, sheet := range golden.GetSheetList() {
		goldenRows, err := golden.GetRows(sheet)
		require.NoError(t, err)
		actualRows, err := actual.GetRows(sheet)
		require.NoError(t, err)

		require.Equalf(t, len(goldenRows), len(actualRows), "sheet %q: row count differs", sheet)

		for r := range goldenRows {
			require.Equalf(t, len(goldenRows[r]), len(actualRows[r]),
				"sheet %q row %d: column count differs", sheet, r+1)
			for c := range goldenRows[r] {
				cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
				require.Equalf(t, goldenRows[r][c], actualRows[r][c],
					"sheet %q cell %s differs", sheet, cell)
			}
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
