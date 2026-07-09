package forex

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// ReferenceRate represents an exchange rate on a specific date
type ReferenceRate struct {
	Date              time.Time
	SourceCurrency    string
	TargetCurrency    string
	TTBuyExchangeRate float64
}

// SBIReferenceRates manages currency exchange Rates
type SBIReferenceRates struct {
	Rates map[string]ReferenceRate // key is date only 2006-01-02
}

// NewSBIReferenceRates creates a new instance
func NewSBIReferenceRates() *SBIReferenceRates {
	return &SBIReferenceRates{
		Rates: make(map[string]ReferenceRate),
	}
}

// LoadFromFile loads Rates from a CSV file
func (s *SBIReferenceRates) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	return s.parseCSV(file)
}

// LoadFromURL loads Rates from GitHub repository
func (s *SBIReferenceRates) LoadFromURL() error {
	url := "https://raw.githubusercontent.com/sahilgupta/sbi-fx-ratekeeper/main/csv_files/SBI_REFERENCE_RATES_USD.csv"

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("fetching Rates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return s.parseCSV(resp.Body)
}

// parseCSV parses the CSV data
func (s *SBIReferenceRates) parseCSV(r io.Reader) error {
	reader := csv.NewReader(r)

	// Skip header
	if _, err := reader.Read(); err != nil {
		return fmt.Errorf("reading header: %w", err)
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading record: %w", err)
		}

		// Parse date (format: "2024-03-31 00:00")
		//dateStr := strings.Split(record[0], " ")[0]
		date, err := time.Parse("2006-01-02 15:04", record[0])
		if err != nil {
			continue // Skip invalid dates
		}

		// Parse TT Buy rate (index 2)
		ttBuy, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			return fmt.Errorf("parsing TT buy rate: %w", err)
		}
		if ttBuy == 0 {
			continue // Skip zero Rates
		}

		s.Rates[date.Format(time.DateOnly)] = ReferenceRate{
			Date:              date,
			SourceCurrency:    "USD",
			TargetCurrency:    "INR",
			TTBuyExchangeRate: ttBuy,
		}
	}

	if len(s.Rates) == 0 {
		return fmt.Errorf("no valid Rates found in CSV")
	}

	return nil
}

// maxRateLookbackDays bounds how far GetRate walks backwards looking for a
// published rate. SBI publishes rates on (almost) every working day, so a gap
// wider than this indicates missing/corrupt reference-rate data rather than a
// normal weekend/holiday, and we fail loudly instead of silently drifting to a
// stale rate far from the requested date.
const maxRateLookbackDays = 15

// GetRate gets the exchange rate for a specific date
func (s *SBIReferenceRates) GetRate(date time.Time, adjustToPreviousMonth bool) (time.Time, ReferenceRate, error) {
	adjustedDate := date

	if adjustToPreviousMonth {
		// Get last day of previous month
		adjustedDate = adjustedDate.AddDate(0, 0, -1*adjustedDate.Day())
	}

	// If exact date not found, go backwards until we find a rate, but only
	// within the lookback window.
	searchFrom := adjustedDate
	for i := 0; i < maxRateLookbackDays; i++ {
		if rate, ok := s.Rates[adjustedDate.Format(time.DateOnly)]; ok {
			return adjustedDate, rate, nil
		}
		adjustedDate = adjustedDate.AddDate(0, 0, -1)
	}

	return time.Time{}, ReferenceRate{}, fmt.Errorf(
		"no exchange rate found within %d days on or before %s (searched %s..%s); SBI reference rate data likely has a gap",
		maxRateLookbackDays,
		searchFrom.Format(time.DateOnly),
		adjustedDate.AddDate(0, 0, 1).Format(time.DateOnly),
		searchFrom.Format(time.DateOnly),
	)
}
