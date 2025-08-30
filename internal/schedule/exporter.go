package schedule

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
	"github.com/xuri/excelize/v2"
)

// ExportToExcel exports all schedules to an Excel file
func ExportToExcel(faSchedule *ForeignAssetsA3, cgSchedule *CapitalGains, alSchedule *AssetsAndLiabilities, outputFile string) error {
	f := excelize.NewFile()

	// Create Schedule FA A3 sheet
	if err := exportScheduleFAA3(f, faSchedule); err != nil {
		return fmt.Errorf("exporting Schedule FA A3: %w", err)
	}

	// Create Schedule CG sheet
	if err := exportScheduleCG(f, cgSchedule); err != nil {
		return fmt.Errorf("exporting Schedule CG: %w", err)
	}

	// Create Schedule AL sheet
	if err := exportScheduleAL(f, alSchedule); err != nil {
		return fmt.Errorf("exporting Schedule AL: %w", err)
	}

	// Delete default sheet
	f.DeleteSheet("Sheet1")

	// Save file
	if err := f.SaveAs(outputFile); err != nil {
		return fmt.Errorf("saving file: %w", err)
	}

	return nil
}

func exportScheduleFAA3(f *excelize.File, schedule *ForeignAssetsA3) error {
	sheetName := "Schedule_FA_A3"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}
	f.SetActiveSheet(index)

	// Headers
	headers := []string{
		"Source File", "Row", "Broker", "Transaction Type", "Award Number",
		"Ticker", "Shares", "Issue Date", "FMV on Issue Date",
		"Sale Date", "FMV on Sale Date",
		"Peak Closing Date", "Peak Closing Value",
		"Year End Date", "Year End Closing Value",
		"Date of Acquiring Interest",
		"Initial Value (INR)", "Peak Value (INR)", "Closing Value (INR)",
		"Gross Amount Paid", "Proceeds from Sale",
	}

	for col, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+col)
		f.SetCellValue(sheetName, cell, header)
	}

	// Data rows
	for i, record := range schedule.Records {
		row := i + 2

		var ticker, shares string
		var issueDate time.Time
		var issueFMV float64
		var saleDate string
		var saleFMV string

		switch r := record.ShareRecord.(type) {
		case stock.ShareIssuedRecord:
			ticker = r.Ticker
			shares = fmt.Sprintf("%d", r.SharesIssued)
			issueDate = r.IssueDate
			issueFMV = r.FMVPerShare
			saleDate = "-"
			saleFMV = "-"
		case stock.ShareSoldRecord:
			ticker = r.Ticker
			shares = fmt.Sprintf("%d", r.SharesSold)
			issueDate = r.IssueDate
			issueFMV = r.FMVOnIssueDate
			saleDate = r.SaleDate.Format("2006-01-02")
			saleFMV = fmt.Sprintf("%.2f", r.FMVOnSaleDate)
		}

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), filepath.Base(record.ShareRecord.GetSourceMetadata().FileName))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), record.ShareRecord.GetSourceMetadata().Row)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), "ETrade")
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), string(record.TransactionType))
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), record.ShareRecord.GetAwardNumber())
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), ticker)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), shares)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), issueDate.Format("2006-01-02"))
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), issueFMV)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), saleDate)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), saleFMV)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), record.PeakClosingDate.Format("2006-01-02"))
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), record.PeakClosingValue)
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), record.LastTradingDate.Format("2006-01-02"))
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), record.YearEndClosingValue)
		f.SetCellValue(sheetName, fmt.Sprintf("P%d", row), record.DateOfAcquiringInterest.Format("2006-01-02"))
		f.SetCellValue(sheetName, fmt.Sprintf("Q%d", row), record.InitialValueOfInvestment)
		f.SetCellValue(sheetName, fmt.Sprintf("R%d", row), record.PeakValueOfInvestment)
		f.SetCellValue(sheetName, fmt.Sprintf("S%d", row), record.ClosingValue)
		f.SetCellValue(sheetName, fmt.Sprintf("T%d", row), record.TotalGrossAmountPaid)
		f.SetCellValue(sheetName, fmt.Sprintf("U%d", row), record.TotalProceedsFromSale)
	}

	return nil
}

func exportScheduleCG(f *excelize.File, schedule *CapitalGains) error {
	sheetName := "Schedule_CG"
	_, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// Headers
	headers := []string{
		"Source File", "Row", "Broker", "Award Number", "Ticker",
		"Shares Sold", "Issue Date", "FMV on Issue Date",
		"Sale Date", "FMV on Sale Date",
		"Cost of Acquisition (INR)", "Cost of Improvement (INR)",
		"Expenditure on Transfer (INR)", "Full Value of Consideration (INR)",
		"Short Term Capital Gain (INR)",
	}

	for col, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+col)
		f.SetCellValue(sheetName, cell, header)
	}

	// Data rows
	for i, record := range schedule.Records {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), filepath.Base(record.ShareRecord.SourceMetadata.FileName))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), record.ShareRecord.SourceMetadata.Row)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), record.ShareRecord.Broker)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), record.ShareRecord.AwardNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), record.ShareRecord.Ticker)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), record.ShareRecord.SharesSold)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), record.ShareRecord.IssueDate.Format("2006-01-02"))
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), record.ShareRecord.FMVOnIssueDate)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), record.ShareRecord.SaleDate.Format("2006-01-02"))
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), record.ShareRecord.FMVOnSaleDate)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), record.CostOfAcquisition)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), record.CostOfImprovement)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), record.ExpenditureOnTransfer)
		f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), record.FullValueOfConsideration)
		f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), record.ShortTermCapitalGain)
	}

	return nil
}

func exportScheduleAL(f *excelize.File, schedule *AssetsAndLiabilities) error {
	sheetName := "Schedule_AL"
	_, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// Headers
	headers := []string{
		"Source File", "Row", "Broker", "Award Number", "Ticker",
		"Shares Issued", "Issue Date", "FMV on Issue Date",
		"Cost of Acquisition (INR)", "Cost of Improvement (INR)",
	}

	for col, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+col)
		f.SetCellValue(sheetName, cell, header)
	}

	// Data rows
	for i, record := range schedule.Records {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), filepath.Base(record.ShareRecord.SourceMetadata.FileName))
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), record.ShareRecord.SourceMetadata.Row)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), record.ShareRecord.Broker)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), record.ShareRecord.AwardNumber)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), record.ShareRecord.Ticker)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), record.ShareRecord.SharesIssued)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), record.ShareRecord.IssueDate.Format("2006-01-02"))
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), record.ShareRecord.FMVPerShare)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), record.CostOfAcquisition)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), record.CostOfImprovement)
	}

	return nil
}
