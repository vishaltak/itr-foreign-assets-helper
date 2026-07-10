package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vtak/itr-foreign-assets-helper/internal/etrade"
	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
	"github.com/vtak/itr-foreign-assets-helper/internal/schedule"
	"github.com/vtak/itr-foreign-assets-helper/internal/stock"
)

func main() {
	var (
		financialYearStr = flag.String("financial-year", "", "Financial year (e.g., 2025-2026)")
		holdingsFile     = flag.String("etrade-holdings", "", "ETrade holdings file (Excel)")
		gainsFile        = flag.String("etrade-sale-transactions", "", "ETrade gains and losses file (Excel)")
		sbiRatesFile     = flag.String("sbi-reference-rates", "", "SBI reference rates CSV (optional)")
		outputDir        = flag.String("output-dir", "output", "Output directory")
	)

	flag.Parse()

	// Validate required flags
	if *financialYearStr == "" || *holdingsFile == "" || *gainsFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	financialYear, err := stock.ParseFinancialYear(*financialYearStr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Processing ITR data for financial year %s", financialYear.String())

	// Load SBI reference rates
	log.Println("Loading SBI reference rates...")
	forexRates := forex.NewSBIReferenceRates()

	if *sbiRatesFile != "" {
		if err := forexRates.LoadFromFile(*sbiRatesFile); err != nil {
			log.Fatalf("Failed to load SBI rates from file: %v", err)
		}
	} else {
		log.Println("Downloading SBI rates from GitHub...")
		if err := forexRates.LoadFromURL(); err != nil {
			log.Fatalf("Failed to download SBI rates: %v", err)
		}
	}

	// Initialize Yahoo Finance client
	yahooClient := stock.NewYahooClient()

	// Initialize ETrade processor
	processor := etrade.NewProcessor(yahooClient, forexRates)

	// Process holdings file
	log.Printf("Processing holdings file: %s", *holdingsFile)
	sharesIssued, err := processor.ProcessHoldings(*holdingsFile)
	if err != nil {
		log.Fatalf("Failed to process holdings: %v", err)
	}
	log.Printf("Found %d share issued records", len(sharesIssued))

	// Process gains and losses file
	log.Printf("Processing gains and losses file: %s", *gainsFile)
	sharesSold, err := processor.ProcessGainsAndLosses(*gainsFile)
	if err != nil {
		log.Fatalf("Failed to process gains and losses: %v", err)
	}
	log.Printf("Found %d share sold records", len(sharesSold))

	// Generate Schedule FA A3
	log.Println("Generating Schedule FA A3...")
	faSchedule, err := schedule.GenerateScheduleFAA3(sharesIssued, sharesSold, yahooClient, forexRates, *financialYear)
	if err != nil {
		log.Fatalf("Failed to generate Schedule FA A3: %v", err)
	}
	log.Printf("Generated %d records for Schedule FA A3", len(faSchedule.Records))

	// Generate Schedule CG
	log.Println("Generating Schedule CG...")
	cgSchedule, err := schedule.GenerateScheduleCG(sharesSold, forexRates, *financialYear)
	if err != nil {
		log.Fatalf("Failed to generate Schedule CG: %v", err)
	}
	log.Printf("Generated %d records for Schedule CG", len(cgSchedule.Records))

	// Generate Schedule AL
	log.Println("Generating Schedule AL...")
	alSchedule, err := schedule.GenerateScheduleAL(sharesIssued, forexRates, *financialYear)
	if err != nil {
		log.Fatalf("Failed to generate Schedule AL: %v", err)
	}
	log.Printf("Generated %d records for Schedule AL", len(alSchedule.Records))

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Export to Excel
	outputFile := filepath.Join(*outputDir, fmt.Sprintf("itr-helper-fy-%s.xlsx", financialYear.String()))
	log.Printf("Exporting to Excel: %s", outputFile)

	if err := schedule.ExportToExcel(faSchedule, cgSchedule, alSchedule, outputFile); err != nil {
		log.Fatalf("Failed to export to Excel: %v", err)
	}

	log.Println("Successfully generated ITR data!")

	// Advise on the rounding effect of ETrade's per-share prices. Proceeds use
	// the reported Total Proceeds; this reports how much the quantity x
	// per-share reconstruction would have differed, in case cross-checking.
	if total, largest := schedule.ProceedsRoundingDiscrepancy(sharesSold, forexRates); total >= 1.0 {
		log.Printf("NOTE: sale proceeds use ETrade's reported Total Proceeds. Valuing them "+
			"as quantity x per-share price instead would differ by ~₹%.2f in aggregate "+
			"(largest ₹%.2f on a single lot), due to ETrade rounding its per-share figures.",
			total, largest)
	}

	// Print summary
	fmt.Println("\n=== SUMMARY ===")
	fmt.Printf("Financial Year: %s\n", financialYear.String())
	fmt.Printf("Shares Issued Records: %d\n", len(sharesIssued))
	fmt.Printf("Shares Sold Records: %d\n", len(sharesSold))
	fmt.Printf("Schedule FA A3 Records: %d\n", len(faSchedule.Records))
	fmt.Printf("Schedule CG Records: %d\n", len(cgSchedule.Records))
	fmt.Printf("Schedule AL Records: %d\n", len(alSchedule.Records))
	fmt.Printf("Output File: %s\n", outputFile)

	fmt.Println("\nIMPORTANT REMINDERS:")
	fmt.Println("1. Schedule FA A2: Add cash balance from December 31 statement manually")
	fmt.Println("2. Schedule AL: Add cash balance from March 31 statement manually")
	fmt.Println("3. Review all generated data before filing")
}
