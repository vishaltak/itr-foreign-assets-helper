package stock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseFinancialYearBoundaries(t *testing.T) {
	fy, err := ParseFinancialYear("2025-2026")
	require.NoError(t, err)

	// Indian financial year runs 1 April to 31 March.
	require.Equal(t, time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), fy.Start)
	require.Equal(t, time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC), fy.End)
}

func TestFinancialYearForeignAssetsCalendarYear(t *testing.T) {
	fy, err := ParseFinancialYear("2025-2026")
	require.NoError(t, err)

	// Schedule FA is reported for the calendar year of the FY start,
	// i.e. 1 January to 31 December (both inclusive).
	require.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), fy.ForeignAssetsStart())
	require.Equal(t, time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), fy.ForeignAssetsEnd())
}
