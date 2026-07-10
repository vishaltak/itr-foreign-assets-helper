package stock

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type FinancialYear struct {
	Start time.Time
	End   time.Time
}

func (y *FinancialYear) String() string {
	return fmt.Sprintf("%d-%d", y.Start.Year(), y.End.Year())
}

// ForeignAssetsStart returns the start of the calendar year used for
// Schedule FA reporting: 1 January of the financial year's start year.
func (y *FinancialYear) ForeignAssetsStart() time.Time {
	return time.Date(y.Start.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
}

// ForeignAssetsEnd returns the end of the calendar year used for
// Schedule FA reporting: 31 December of the financial year's start year.
func (y *FinancialYear) ForeignAssetsEnd() time.Time {
	return time.Date(y.Start.Year(), time.December, 31, 0, 0, 0, 0, time.UTC)
}

func ParseFinancialYear(value string) (*FinancialYear, error) {
	years := strings.Split(value, "-")
	if len(years) != 2 {
		return nil, errors.New("invalid financial year")
	}
	start, err := time.Parse("2006", years[0])
	if err != nil {
		return nil, errors.New("invalid financial year")
	}
	end, err := time.Parse("2006", years[1])
	if err != nil {
		return nil, errors.New("invalid financial year")
	}
	if end.Year() != start.Year()+1 {
		return nil, errors.New("invalid financial year: end year must be start year + 1")
	}
	// The Indian financial year runs from 1 April of the start year to
	// 31 March of the end year.
	return &FinancialYear{
		Start: time.Date(start.Year(), time.April, 1, 0, 0, 0, 0, time.UTC),
		End:   time.Date(end.Year(), time.March, 31, 0, 0, 0, 0, time.UTC),
	}, nil
}

// PriceData represents stock price for a specific date
type PriceData struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

type ShareRecord interface {
	GetSourceMetadata() SourceMetadata
	GetTicker() string
	GetAwardNumber() string
	GetComments() string
}

// ShareIssuedRecord represents shares issued (from holdings file)
type ShareIssuedRecord struct {
	SourceMetadata SourceMetadata
	Broker         string
	Ticker         string
	AwardNumber    string
	SharesIssued   float64
	IssueDate      time.Time
	FMVOnIssueDate float64
	Comments       string
}

func (s ShareIssuedRecord) GetSourceMetadata() SourceMetadata {
	return s.SourceMetadata
}

func (s ShareIssuedRecord) GetTicker() string {
	return s.Ticker
}

func (s ShareIssuedRecord) GetAwardNumber() string {
	return s.AwardNumber
}

func (s ShareIssuedRecord) GetComments() string {
	return s.Comments
}

// ShareSoldRecord represents shares sold (from gains/losses file)
type ShareSoldRecord struct {
	SourceMetadata  SourceMetadata
	Broker          string
	Ticker          string
	AwardNumber     string
	IssueDate       time.Time
	FMVOnIssueDate  float64
	SharesSold      float64
	SaleDate        time.Time
	FMVOnSaleDate   float64
	SaleOrderNumber string
	Comments        string
}

func (s ShareSoldRecord) GetSourceMetadata() SourceMetadata {
	return s.SourceMetadata
}

func (s ShareSoldRecord) GetTicker() string {
	return s.Ticker
}

func (s ShareSoldRecord) GetAwardNumber() string {
	return s.AwardNumber
}

func (s ShareSoldRecord) GetComments() string {
	return s.Comments
}

// SourceMetadata tracks where data came from
type SourceMetadata struct {
	FileName  string
	SheetName string
	Row       int
}

// TransactionType enum
type TransactionType string

const (
	TransactionIssued TransactionType = "issued"
	TransactionSold   TransactionType = "sold"
)
