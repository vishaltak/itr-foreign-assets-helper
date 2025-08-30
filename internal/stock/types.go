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
	return &FinancialYear{start, end}, nil
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
	GetAwardNumber() int
	GetComments() string
}

// ShareIssuedRecord represents shares issued (from holdings file)
type ShareIssuedRecord struct {
	SourceMetadata SourceMetadata
	Broker         string
	Ticker         string
	AwardNumber    int
	SharesIssued   int
	IssueDate      time.Time
	FMVPerShare    float64
	Comments       string
}

func (s ShareIssuedRecord) GetSourceMetadata() SourceMetadata {
	return s.SourceMetadata
}

func (s ShareIssuedRecord) GetTicker() string {
	return s.Ticker
}

func (s ShareIssuedRecord) GetAwardNumber() int {
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
	AwardNumber     int
	IssueDate       time.Time
	FMVOnIssueDate  float64
	SharesSold      int
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

func (s ShareSoldRecord) GetAwardNumber() int {
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
