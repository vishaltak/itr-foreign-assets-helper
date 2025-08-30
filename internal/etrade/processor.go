package etrade

import (
	"fmt"
	"strings"
	"time"

	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
	"github.com/xuri/excelize/v2"
)

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

	if len(rows) < 2 {
		return nil, fmt.Errorf("insufficient data in holdings file")
	}

	// Find column indices
	headers := rows[0]
	colMap := make(map[string]int)
	for i, header := range headers {
		colMap[header] = i
	}

	requiredCols := map[string]string{
		"Symbol":            "ticker",
		"Sellable Qty.":     "shares",
		"Grant Number":      "award",
		"Release Date":      "date",
		"Purchase Date FMV": "fmv",
	}

	// Verify required columns exist
	for col := range requiredCols {
		if _, ok := colMap[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	var records []stock.ShareIssuedRecord

	// Process data rows (skip header at index 0 and total at last row)
	for i := 1; i < len(rows)-1; i++ {
		row := rows[i]
		if len(row) <= colMap["Purchase Date FMV"] {
			continue // Skip incomplete rows
		}

		// Parse release date (format: "15-Mar-2024")
		dateStr := row[colMap["Release Date"]]
		issueDate, err := time.Parse("02-Jan-2006", dateStr)
		if err != nil {
			return nil, fmt.Errorf("parsing date %s at row %d: %w", dateStr, i+1, err)
		}

		// Parse FMV (remove $ sign)
		fmvStr := strings.TrimPrefix(row[colMap["Purchase Date FMV"]], "$")
		fmvStr = strings.ReplaceAll(fmvStr, ",", "")
		var fmv float64
		fmt.Sscanf(fmvStr, "%f", &fmv)

		// Parse shares
		var shares int
		fmt.Sscanf(row[colMap["Sellable Qty."]], "%d", &shares)

		// Parse award number
		var awardNum int
		fmt.Sscanf(row[colMap["Grant Number"]], "%d", &awardNum)

		record := stock.ShareIssuedRecord{
			SourceMetadata: stock.SourceMetadata{
				FileName:  filename,
				SheetName: sheetName,
				Row:       i + 1,
			},
			Broker:       "ETrade",
			Ticker:       row[colMap["Symbol"]],
			AwardNumber:  awardNum,
			SharesIssued: shares,
			IssueDate:    issueDate,
			FMVPerShare:  fmv,
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

	if len(rows) < 3 {
		return nil, fmt.Errorf("insufficient data in gains file")
	}

	// Find column indices (header is at row 0)
	headers := rows[0]
	colMap := make(map[string]int)
	for i, header := range headers {
		colMap[header] = i
	}

	requiredCols := map[string]string{
		"Symbol":             "ticker",
		"Quantity":           "shares",
		"Grant Number":       "award",
		"Date Acquired":      "issue_date",
		"Vest Date FMV":      "issue_fmv",
		"Date Sold":          "sale_date",
		"Proceeds Per Share": "sale_fmv",
		"Order Number":       "order",
	}

	// Verify required columns
	for col := range requiredCols {
		if _, ok := colMap[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	var records []stock.ShareSoldRecord

	// Process data rows (skip header at 0, summary at 1)
	for i := 2; i < len(rows); i++ {
		row := rows[i]
		if len(row) <= colMap["Order Number"] {
			continue // Skip incomplete rows
		}

		// Parse dates (format: "MM/DD/YYYY")
		issueDateStr := row[colMap["Date Acquired"]]
		issueDate, err := time.Parse("01/02/2006", issueDateStr)
		if err != nil {
			return nil, fmt.Errorf("parsing issue date %s at row %d: %w", issueDateStr, i+1, err)
		}

		saleDateStr := row[colMap["Date Sold"]]
		saleDate, err := time.Parse("01/02/2006", saleDateStr)
		if err != nil {
			return nil, fmt.Errorf("parsing sale date %s at row %d: %w", saleDateStr, i+1, err)
		}

		// Parse FMV values
		issueFMVStr := strings.TrimPrefix(row[colMap["Vest Date FMV"]], "$")
		issueFMVStr = strings.ReplaceAll(issueFMVStr, ",", "")
		var issueFMV float64
		fmt.Sscanf(issueFMVStr, "%f", &issueFMV)

		saleFMVStr := strings.TrimPrefix(row[colMap["Proceeds Per Share"]], "$")
		saleFMVStr = strings.ReplaceAll(saleFMVStr, ",", "")
		var saleFMV float64
		fmt.Sscanf(saleFMVStr, "%f", &saleFMV)

		// Parse quantities
		var shares int
		fmt.Sscanf(row[colMap["Quantity"]], "%d", &shares)

		var awardNum int
		fmt.Sscanf(row[colMap["Grant Number"]], "%d", &awardNum)

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
			SaleOrderNumber: row[colMap["Order Number"]],
		}

		records = append(records, record)
	}

	return records, nil
}
