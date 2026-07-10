package etrade

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
	"github.com/xuri/excelize/v2"
)

const (
	// holdingsFooterRows is the number of trailing total rows in the ETrade
	// holdings "Sellable" sheet: "Options Total", "SARS Total",
	// "Shares Blocked total" and "Overall Total".
	holdingsFooterRows = 4

	// gainsHeaderRows is the number of leading rows to skip in the ETrade gains
	// "G&L_Expanded" sheet: the header row and a "Summary" row.
	gainsHeaderRows = 2
)

// parseMoney parses a currency value, stripping a leading "$" and any
// thousands separators (e.g. "$1,234.50" -> 1234.50).
func parseMoney(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	s = strings.ReplaceAll(s, ",", "")
	return strconv.ParseFloat(s, 64)
}

// parseShares parses a (possibly fractional) share quantity.
func parseShares(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	return strconv.ParseFloat(s, 64)
}

// parseDate parses a date cell, returning an actionable error that names the
// column and the expected text format when parsing fails (typically because the
// column was exported as a real date cell rather than text).
func parseDate(column, layout, example, value string, row int) (time.Time, error) {
	t, err := time.Parse(layout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"parsing %s %q at row %d: expected a text date like %q (format the column as text if it is a date cell): %w",
			column, value, row, example, err,
		)
	}
	return t, nil
}

// buildColumnMap maps header names to their column index and fails loud if any
// required column is duplicated (a mislabeled sheet) or missing.
func buildColumnMap(headers []string, required map[string]string) (map[string]int, error) {
	colMap := make(map[string]int)
	seen := make(map[string]bool)
	for i, header := range headers {
		if _, ok := required[header]; ok {
			if seen[header] {
				return nil, fmt.Errorf("duplicate column %q in sheet", header)
			}
			seen[header] = true
		}
		colMap[header] = i
	}
	for col := range required {
		if _, ok := colMap[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}
	return colMap, nil
}

// Processor handles ETrade file processing
type Processor struct {
	yahooClient *stock.YahooClient
	forexRates  *forex.SBIReferenceRates
}

// NewProcessor creates a new ETrade processor
func NewProcessor(yahooClient *stock.YahooClient, forexRates *forex.SBIReferenceRates) *Processor {
	return &Processor{
		yahooClient: yahooClient,
		forexRates:  forexRates,
	}
}

// ProcessHoldings processes the ETrade holdings file
func (p *Processor) ProcessHoldings(filename string) ([]stock.ShareIssuedRecord, error) {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	sheetName := "Sellable"
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("reading sheet %s: %w", sheetName, err)
	}

	if len(rows) <= holdingsFooterRows {
		return nil, fmt.Errorf("insufficient data in holdings file")
	}

	requiredCols := map[string]string{
		"Symbol":            "ticker",
		"Sellable Qty.":     "shares",
		"Grant Number":      "award",
		"Release Date":      "date",
		"Purchase Date FMV": "fmv",
	}

	colMap, err := buildColumnMap(rows[0], requiredCols)
	if err != nil {
		return nil, err
	}

	var records []stock.ShareIssuedRecord

	// Process data rows: skip the header (row 0) and the trailing total rows.
	for i := 1; i < len(rows)-holdingsFooterRows; i++ {
		row := rows[i]
		if len(row) <= colMap["Purchase Date FMV"] {
			continue // Skip incomplete rows
		}

		// Parse release date (format: "15-Mar-2024")
		issueDate, err := parseDate("Release Date", "02-Jan-2006", "15-Mar-2024", row[colMap["Release Date"]], i+1)
		if err != nil {
			return nil, err
		}

		// Parse FMV (remove $ sign)
		fmv, err := parseMoney(row[colMap["Purchase Date FMV"]])
		if err != nil {
			return nil, fmt.Errorf("parsing FMV at row %d: %w", i+1, err)
		}

		// Parse shares (may be fractional)
		shares, err := parseShares(row[colMap["Sellable Qty."]])
		if err != nil {
			return nil, fmt.Errorf("parsing shares at row %d: %w", i+1, err)
		}

		// Award/grant number is an identifier, kept as a string.
		awardNum := strings.TrimSpace(row[colMap["Grant Number"]])

		record := stock.ShareIssuedRecord{
			SourceMetadata: stock.SourceMetadata{
				FileName:  filename,
				SheetName: sheetName,
				Row:       i + 1,
			},
			Broker:         "ETrade",
			Ticker:         row[colMap["Symbol"]],
			AwardNumber:    awardNum,
			SharesIssued:   shares,
			IssueDate:      issueDate,
			FMVOnIssueDate: fmv,
		}

		records = append(records, record)
	}

	return records, nil
}

// ProcessGainsAndLosses processes the gains and losses file
func (p *Processor) ProcessGainsAndLosses(filename string) ([]stock.ShareSoldRecord, error) {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	sheetName := "G&L_Expanded"
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("reading sheet %s: %w", sheetName, err)
	}

	if len(rows) < gainsHeaderRows {
		return nil, fmt.Errorf("insufficient data in gains file")
	}

	requiredCols := map[string]string{
		"Symbol":             "ticker",
		"Quantity":           "shares",
		"Grant Number":       "award",
		"Date Acquired":      "issue_date",
		"Vest Date FMV":      "issue_fmv",
		"Date Sold":          "sale_date",
		"Proceeds Per Share": "sale_fmv",
		"Total Proceeds":     "total_proceeds",
		"Order Number":       "order",
	}

	colMap, err := buildColumnMap(rows[0], requiredCols)
	if err != nil {
		return nil, err
	}

	var records []stock.ShareSoldRecord

	// Process data rows: skip the header (row 0) and the summary row(s).
	for i := gainsHeaderRows; i < len(rows); i++ {
		row := rows[i]
		if len(row) <= colMap["Order Number"] {
			continue // Skip incomplete rows
		}

		// Parse dates (format: "MM/DD/YYYY")
		issueDate, err := parseDate("Date Acquired", "01/02/2006", "03/15/2023", row[colMap["Date Acquired"]], i+1)
		if err != nil {
			return nil, err
		}

		saleDate, err := parseDate("Date Sold", "01/02/2006", "06/15/2024", row[colMap["Date Sold"]], i+1)
		if err != nil {
			return nil, err
		}

		// Parse FMV values
		issueFMV, err := parseMoney(row[colMap["Vest Date FMV"]])
		if err != nil {
			return nil, fmt.Errorf("parsing issue FMV at row %d: %w", i+1, err)
		}

		saleFMV, err := parseMoney(row[colMap["Proceeds Per Share"]])
		if err != nil {
			return nil, fmt.Errorf("parsing sale FMV at row %d: %w", i+1, err)
		}

		totalProceeds, err := parseMoney(row[colMap["Total Proceeds"]])
		if err != nil {
			return nil, fmt.Errorf("parsing total proceeds at row %d: %w", i+1, err)
		}

		// Parse quantities (may be fractional)
		shares, err := parseShares(row[colMap["Quantity"]])
		if err != nil {
			return nil, fmt.Errorf("parsing shares at row %d: %w", i+1, err)
		}

		// Award/grant number is an identifier, kept as a string.
		awardNum := strings.TrimSpace(row[colMap["Grant Number"]])

		record := stock.ShareSoldRecord{
			SourceMetadata: stock.SourceMetadata{
				FileName:  filename,
				SheetName: sheetName,
				Row:       i + 1,
			},
			Broker:          "ETrade",
			Ticker:          row[colMap["Symbol"]],
			AwardNumber:     awardNum,
			IssueDate:       issueDate,
			FMVOnIssueDate:  issueFMV,
			SharesSold:      shares,
			SaleDate:        saleDate,
			FMVOnSaleDate:   saleFMV,
			TotalProceeds:   totalProceeds,
			SaleOrderNumber: row[colMap["Order Number"]],
		}

		records = append(records, record)
	}

	return records, nil
}
