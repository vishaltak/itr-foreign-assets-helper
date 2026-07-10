package schedule

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
	"github.com/xuri/excelize/v2"
)

// sheetStyles holds the reusable cell-style IDs for a workbook.
type sheetStyles struct {
	text    int
	integer int
	usd     int
	inr     int
	date    int
	header  int
}

func newSheetStyles(f *excelize.File) (*sheetStyles, error) {
	fmtStr := func(s string) (int, error) {
		return f.NewStyle(&excelize.Style{CustomNumFmt: &s})
	}
	s := &sheetStyles{}
	var err error
	if s.text, err = fmtStr("@"); err != nil {
		return nil, err
	}
	if s.integer, err = fmtStr("0"); err != nil {
		return nil, err
	}
	if s.usd, err = fmtStr(`"$"#,##0.00`); err != nil {
		return nil, err
	}
	if s.inr, err = fmtStr(`"₹"#,##0.00`); err != nil {
		return nil, err
	}
	if s.date, err = fmtStr("yyyy-mm-dd"); err != nil {
		return nil, err
	}
	if s.header, err = f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}}); err != nil {
		return nil, err
	}
	return s, nil
}

// colSpec describes one output column: its header, cell style (0 = default,
// e.g. share counts), and how to extract the value for a record. extract
// returns nil to leave the cell blank.
type colSpec[R any] struct {
	header  string
	style   int
	extract func(R) any
}

func baseName(path string) string { return filepath.Base(path) }

// dateCell returns a date-only cell value (so date columns render cleanly as
// dates, not date-times), or nil for a zero time.
func dateCell(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// Cell accessors for a ValuationDate. A zero ValuationDate (Date unset)
// yields blank cells for all three of its columns.
func (d ValuationDate) eventDateCell() any { return dateCell(d.Date) }
func (d ValuationDate) rateDateCell() any {
	if d.Date.IsZero() {
		return nil
	}
	return dateCell(d.Rate.Date)
}
func (d ValuationDate) rateCell() any {
	if d.Date.IsZero() {
		return nil
	}
	return d.Rate.TTBuyExchangeRate
}

// writeSheet creates a sheet and writes the header row plus one row per record.
// Only formatting is applied via styles; underlying values are written as-is.
func writeSheet[R any](f *excelize.File, sheet string, records []R, cols []colSpec[R], headerStyle int) error {
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}

	for i, c := range cols {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, c.header); err != nil {
			return err
		}
	}

	for ri, rec := range records {
		row := ri + 2
		for ci, c := range cols {
			v := c.extract(rec)
			if v == nil {
				continue
			}
			cell, _ := excelize.CoordinatesToCellName(ci+1, row)
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				return err
			}
		}
	}

	// Bold header row.
	lastCol, _ := excelize.ColumnNumberToName(len(cols))
	if err := f.SetCellStyle(sheet, "A1", fmt.Sprintf("%s1", lastCol), headerStyle); err != nil {
		return err
	}
	// Apply each column's style over its data range (overrides any auto-styling
	// excelize applies when writing typed values such as dates).
	if len(records) > 0 {
		for ci, c := range cols {
			if c.style == 0 {
				continue
			}
			col, _ := excelize.ColumnNumberToName(ci + 1)
			if err := f.SetCellStyle(sheet, fmt.Sprintf("%s2", col), fmt.Sprintf("%s%d", col, len(records)+1), c.style); err != nil {
				return err
			}
		}
	}
	return nil
}

// ExportToExcel exports all schedules to an Excel file.
func ExportToExcel(faSchedule *ForeignAssetsA3, cgSchedule *CapitalGains, alSchedule *AssetsAndLiabilities, outputFile string) error {
	f := excelize.NewFile()

	s, err := newSheetStyles(f)
	if err != nil {
		return fmt.Errorf("creating styles: %w", err)
	}

	if err := writeSheet(f, "Schedule_FA_A3", faSchedule.Records, faColumns(s), s.header); err != nil {
		return fmt.Errorf("exporting Schedule FA A3: %w", err)
	}
	f.SetActiveSheet(0)
	if err := writeSheet(f, "Schedule_CG", cgSchedule.Records, cgColumns(s), s.header); err != nil {
		return fmt.Errorf("exporting Schedule CG: %w", err)
	}
	if err := writeSheet(f, "Schedule_AL", alSchedule.Records, alColumns(s), s.header); err != nil {
		return fmt.Errorf("exporting Schedule AL: %w", err)
	}

	f.DeleteSheet("Sheet1")

	if err := f.SaveAs(outputFile); err != nil {
		return fmt.Errorf("saving file: %w", err)
	}
	return nil
}

// --- Schedule FA A3 columns ---

func faBroker(r ForeignAssetsA3Record) string {
	switch s := r.ShareRecord.(type) {
	case stock.ShareIssuedRecord:
		return s.Broker
	case stock.ShareSoldRecord:
		return s.Broker
	}
	return ""
}

func faShares(r ForeignAssetsA3Record) any {
	switch s := r.ShareRecord.(type) {
	case stock.ShareIssuedRecord:
		return s.SharesIssued
	case stock.ShareSoldRecord:
		return s.SharesSold
	}
	return nil
}

func faIssueFMV(r ForeignAssetsA3Record) any {
	switch s := r.ShareRecord.(type) {
	case stock.ShareIssuedRecord:
		return s.FMVOnIssueDate
	case stock.ShareSoldRecord:
		return s.FMVOnIssueDate
	}
	return nil
}

func faSaleFMV(r ForeignAssetsA3Record) any {
	if sr, ok := r.ShareRecord.(stock.ShareSoldRecord); ok {
		return sr.FMVOnSaleDate
	}
	return nil
}

func faColumns(s *sheetStyles) []colSpec[ForeignAssetsA3Record] {
	return []colSpec[ForeignAssetsA3Record]{
		{"Source File", s.text, func(r ForeignAssetsA3Record) any { return baseName(r.ShareRecord.GetSourceMetadata().FileName) }},
		{"Sheet", s.text, func(r ForeignAssetsA3Record) any { return r.ShareRecord.GetSourceMetadata().SheetName }},
		{"Row", s.integer, func(r ForeignAssetsA3Record) any { return r.ShareRecord.GetSourceMetadata().Row }},
		{"Broker", s.text, func(r ForeignAssetsA3Record) any { return faBroker(r) }},
		{"Transaction Type", s.text, func(r ForeignAssetsA3Record) any { return string(r.TransactionType) }},
		{"Award Number", s.text, func(r ForeignAssetsA3Record) any { return r.ShareRecord.GetAwardNumber() }},
		{"Ticker", s.text, func(r ForeignAssetsA3Record) any { return r.ShareRecord.GetTicker() }},
		{"Shares", 0, faShares},
		{"Issue Date", s.date, func(r ForeignAssetsA3Record) any { return r.IssueDate.eventDateCell() }},
		{"FMV on Issue Date", s.usd, faIssueFMV},
		{"TT Buy Rate Date Considered for Issue Date", s.date, func(r ForeignAssetsA3Record) any { return r.IssueDate.rateDateCell() }},
		{"TT Buy Rate Considered for Issue Date", s.inr, func(r ForeignAssetsA3Record) any { return r.IssueDate.rateCell() }},
		{"Sale Date", s.date, func(r ForeignAssetsA3Record) any { return r.SaleDate.eventDateCell() }},
		{"FMV on Sale Date", s.usd, faSaleFMV},
		{"TT Buy Rate Date Considered for Sale Date", s.date, func(r ForeignAssetsA3Record) any { return r.SaleDate.rateDateCell() }},
		{"TT Buy Rate Considered for Sale Date", s.inr, func(r ForeignAssetsA3Record) any { return r.SaleDate.rateCell() }},
		{"Peak Closing Date", s.date, func(r ForeignAssetsA3Record) any { return r.PeakClose.eventDateCell() }},
		{"Peak Closing Value", s.usd, func(r ForeignAssetsA3Record) any { return r.PeakClosingValue }},
		{"TT Buy Rate Date Considered for Peak Closing Date", s.date, func(r ForeignAssetsA3Record) any { return r.PeakClose.rateDateCell() }},
		{"TT Buy Rate Considered for Peak Closing Date", s.inr, func(r ForeignAssetsA3Record) any { return r.PeakClose.rateCell() }},
		{"Year End Date", s.date, func(r ForeignAssetsA3Record) any { return r.YearEnd.eventDateCell() }},
		{"Year End Closing Value", s.usd, func(r ForeignAssetsA3Record) any {
			if r.YearEnd.Date.IsZero() {
				return nil
			}
			return r.YearEndClosingValue
		}},
		{"TT Buy Rate Date Considered for Year End Date", s.date, func(r ForeignAssetsA3Record) any { return r.YearEnd.rateDateCell() }},
		{"TT Buy Rate Considered for Year End Date", s.inr, func(r ForeignAssetsA3Record) any { return r.YearEnd.rateCell() }},
		{"", 0, func(ForeignAssetsA3Record) any { return nil }}, // spacer
		{"Date of Acquiring Interest", s.date, func(r ForeignAssetsA3Record) any { return r.IssueDate.eventDateCell() }},
		{"Initial Value of Investment", s.inr, func(r ForeignAssetsA3Record) any { return r.InitialValueOfInvestment }},
		{"Peak Value of Investment", s.inr, func(r ForeignAssetsA3Record) any { return r.PeakValueOfInvestment }},
		{"Closing Value", s.inr, func(r ForeignAssetsA3Record) any { return r.ClosingValue }},
		{"Total gross amount paid/credited with respect to the holding during the period", s.inr, func(r ForeignAssetsA3Record) any { return r.TotalGrossAmountPaid }},
		{"Total gross proceeds from sale or redemption of investment during the period", s.inr, func(r ForeignAssetsA3Record) any { return r.TotalProceedsFromSale }},
	}
}

// --- Schedule CG columns ---

func cgColumns(s *sheetStyles) []colSpec[CapitalGainsRecord] {
	return []colSpec[CapitalGainsRecord]{
		{"Source File", s.text, func(r CapitalGainsRecord) any { return baseName(r.ShareRecord.SourceMetadata.FileName) }},
		{"Sheet", s.text, func(r CapitalGainsRecord) any { return r.ShareRecord.SourceMetadata.SheetName }},
		{"Row", s.integer, func(r CapitalGainsRecord) any { return r.ShareRecord.SourceMetadata.Row }},
		{"Broker", s.text, func(r CapitalGainsRecord) any { return r.ShareRecord.Broker }},
		{"Transaction Type", s.text, func(r CapitalGainsRecord) any { return string(stock.TransactionSold) }},
		{"Award Number", s.text, func(r CapitalGainsRecord) any { return r.ShareRecord.AwardNumber }},
		{"Ticker", s.text, func(r CapitalGainsRecord) any { return r.ShareRecord.Ticker }},
		{"Shares Sold", 0, func(r CapitalGainsRecord) any { return r.ShareRecord.SharesSold }},
		{"Issue Date", s.date, func(r CapitalGainsRecord) any { return r.IssueDate.eventDateCell() }},
		{"FMV on Issue Date", s.usd, func(r CapitalGainsRecord) any { return r.ShareRecord.FMVOnIssueDate }},
		{"TT Buy Rate Date Considered for Issue Date", s.date, func(r CapitalGainsRecord) any { return r.IssueDate.rateDateCell() }},
		{"TT Buy Rate Considered for Issue Date", s.inr, func(r CapitalGainsRecord) any { return r.IssueDate.rateCell() }},
		{"Sale Date", s.date, func(r CapitalGainsRecord) any { return r.SaleDate.eventDateCell() }},
		{"FMV on Sale Date", s.usd, func(r CapitalGainsRecord) any { return r.ShareRecord.FMVOnSaleDate }},
		{"TT Buy Rate Date Considered for Sale Date", s.date, func(r CapitalGainsRecord) any { return r.SaleDate.rateDateCell() }},
		{"TT Buy Rate Considered for Sale Date", s.inr, func(r CapitalGainsRecord) any { return r.SaleDate.rateCell() }},
		{"", 0, func(CapitalGainsRecord) any { return nil }}, // spacer
		{"Cost of Acquisition without indexation", s.inr, func(r CapitalGainsRecord) any { return r.CostOfAcquisition }},
		{"Cost of Improvement without indexation", s.inr, func(r CapitalGainsRecord) any { return r.CostOfImprovement }},
		{"Expenditure wholly and exclusively in connection with transfer", s.inr, func(r CapitalGainsRecord) any { return r.ExpenditureOnTransfer }},
		{"Full value of consideration received/receivable", s.inr, func(r CapitalGainsRecord) any { return r.FullValueOfConsideration }},
		{"Short Term Capital Gain", s.inr, func(r CapitalGainsRecord) any { return r.ShortTermCapitalGain }},
	}
}

// --- Schedule AL columns ---

func alColumns(s *sheetStyles) []colSpec[AssetsAndLiabilitiesRecord] {
	return []colSpec[AssetsAndLiabilitiesRecord]{
		{"Source File", s.text, func(r AssetsAndLiabilitiesRecord) any { return baseName(r.ShareRecord.SourceMetadata.FileName) }},
		{"Sheet", s.text, func(r AssetsAndLiabilitiesRecord) any { return r.ShareRecord.SourceMetadata.SheetName }},
		{"Row", s.integer, func(r AssetsAndLiabilitiesRecord) any { return r.ShareRecord.SourceMetadata.Row }},
		{"Broker", s.text, func(r AssetsAndLiabilitiesRecord) any { return r.ShareRecord.Broker }},
		{"Transaction Type", s.text, func(r AssetsAndLiabilitiesRecord) any { return string(stock.TransactionIssued) }},
		{"Award Number", s.text, func(r AssetsAndLiabilitiesRecord) any { return r.ShareRecord.AwardNumber }},
		{"Ticker", s.text, func(r AssetsAndLiabilitiesRecord) any { return r.ShareRecord.Ticker }},
		{"Shares Issued", 0, func(r AssetsAndLiabilitiesRecord) any { return r.ShareRecord.SharesIssued }},
		{"Issue Date", s.date, func(r AssetsAndLiabilitiesRecord) any { return r.IssueDate.eventDateCell() }},
		{"FMV on Issue Date", s.usd, func(r AssetsAndLiabilitiesRecord) any { return r.ShareRecord.FMVOnIssueDate }},
		{"TT Buy Rate Date Considered for Issue Date", s.date, func(r AssetsAndLiabilitiesRecord) any { return r.IssueDate.rateDateCell() }},
		{"TT Buy Rate Considered for Issue Date", s.inr, func(r AssetsAndLiabilitiesRecord) any { return r.IssueDate.rateCell() }},
		{"", 0, func(AssetsAndLiabilitiesRecord) any { return nil }}, // spacer
		{"Cost of Acquisition without indexation", s.inr, func(r AssetsAndLiabilitiesRecord) any { return r.CostOfAcquisition }},
	}
}
