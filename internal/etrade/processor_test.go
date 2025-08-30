package etrade

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
	"github.com/xuri/excelize/v2"
)

func createTestHoldingsFile(t *testing.T) string {
	f := excelize.NewFile()

	sheetName := "Sellable"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		t.Fatal(err)
	}
	f.SetActiveSheet(index)

	// Headers
	f.SetCellValue(sheetName, "A1", "Symbol")
	f.SetCellValue(sheetName, "B1", "Sellable Qty.")
	f.SetCellValue(sheetName, "C1", "Grant Number")
	f.SetCellValue(sheetName, "D1", "Release Date")
	f.SetCellValue(sheetName, "E1", "Purchase Date FMV")

	// Data row
	f.SetCellValue(sheetName, "A2", "AAPL")
	f.SetCellValue(sheetName, "B2", 100)
	f.SetCellValue(sheetName, "C2", "12345")
	f.SetCellValue(sheetName, "D2", "15-Mar-2024")
	f.SetCellValue(sheetName, "E2", "$150.00")

	// Total row (should be skipped)
	f.SetCellValue(sheetName, "A3", "Total")
	f.SetCellValue(sheetName, "B3", 100)

	// Delete default sheet
	f.DeleteSheet("Sheet1")

	// Save to temp file
	tmpFile := filepath.Join(t.TempDir(), "test_holdings.xlsx")
	if err := f.SaveAs(tmpFile); err != nil {
		t.Fatal(err)
	}

	return tmpFile
}

func createTestGainsFile(t *testing.T) string {
	f := excelize.NewFile()

	sheetName := "G&L_Expanded"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		t.Fatal(err)
	}
	f.SetActiveSheet(index)

	// Headers
	f.SetCellValue(sheetName, "A1", "Symbol")
	f.SetCellValue(sheetName, "B1", "Quantity")
	f.SetCellValue(sheetName, "C1", "Grant Number")
	f.SetCellValue(sheetName, "D1", "Date Acquired")
	f.SetCellValue(sheetName, "E1", "Vest Date FMV")
	f.SetCellValue(sheetName, "F1", "Date Sold")
	f.SetCellValue(sheetName, "G1", "Proceeds Per Share")
	f.SetCellValue(sheetName, "H1", "Order Number")

	// Summary row (should be skipped)
	f.SetCellValue(sheetName, "A2", "Summary")

	// Data row
	f.SetCellValue(sheetName, "A3", "AAPL")
	f.SetCellValue(sheetName, "B3", 50)
	f.SetCellValue(sheetName, "C3", "12345")
	f.SetCellValue(sheetName, "D3", "03/15/2023")
	f.SetCellValue(sheetName, "E3", "$140.00")
	f.SetCellValue(sheetName, "F3", "06/15/2024")
	f.SetCellValue(sheetName, "G3", "$180.00")
	f.SetCellValue(sheetName, "H3", "ORD123")

	// Delete default sheet
	f.DeleteSheet("Sheet1")

	// Save to temp file
	tmpFile := filepath.Join(t.TempDir(), "test_gains.xlsx")
	if err := f.SaveAs(tmpFile); err != nil {
		t.Fatal(err)
	}

	return tmpFile
}

func TestProcessor_ProcessHoldings(t *testing.T) {
	holdingsFile := createTestHoldingsFile(t)
	defer os.Remove(holdingsFile)

	yahooClient := stock.NewYahooClient()
	forexRates := forex.NewSBIReferenceRates()

	processor := NewProcessor(yahooClient, forexRates)

	records, err := processor.ProcessHoldings(holdingsFile)
	if err != nil {
		t.Fatalf("ProcessHoldings() error = %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(records))
	}

	if len(records) > 0 {
		record := records[0]

		if record.Ticker != "AAPL" {
			t.Errorf("Expected ticker AAPL, got %s", record.Ticker)
		}

		if record.SharesIssued != 100 {
			t.Errorf("Expected 100 shares, got %d", record.SharesIssued)
		}

		if record.FMVPerShare != 150.00 {
			t.Errorf("Expected FMV 150.00, got %f", record.FMVPerShare)
		}

		expectedDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
		if !record.IssueDate.Equal(expectedDate) {
			t.Errorf("Expected issue date %v, got %v", expectedDate, record.IssueDate)
		}
	}
}

func TestProcessor_ProcessGainsAndLosses(t *testing.T) {
	gainsFile := createTestGainsFile(t)
	defer os.Remove(gainsFile)

	yahooClient := stock.NewYahooClient()
	forexRates := forex.NewSBIReferenceRates()

	processor := NewProcessor(yahooClient, forexRates)

	records, err := processor.ProcessGainsAndLosses(gainsFile)
	if err != nil {
		t.Fatalf("ProcessGainsAndLosses() error = %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(records))
	}

	if len(records) > 0 {
		record := records[0]

		if record.Ticker != "AAPL" {
			t.Errorf("Expected ticker AAPL, got %s", record.Ticker)
		}

		if record.SharesSold != 50 {
			t.Errorf("Expected 50 shares sold, got %d", record.SharesSold)
		}

		if record.FMVOnIssueDate != 140.00 {
			t.Errorf("Expected issue FMV 140.00, got %f", record.FMVOnIssueDate)
		}

		if record.FMVOnSaleDate != 180.00 {
			t.Errorf("Expected sale FMV 180.00, got %f", record.FMVOnSaleDate)
		}

		expectedIssueDate := time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC)
		if !record.IssueDate.Equal(expectedIssueDate) {
			t.Errorf("Expected issue date %v, got %v", expectedIssueDate, record.IssueDate)
		}

		expectedSaleDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		if !record.SaleDate.Equal(expectedSaleDate) {
			t.Errorf("Expected sale date %v, got %v", expectedSaleDate, record.SaleDate)
		}
	}
}
