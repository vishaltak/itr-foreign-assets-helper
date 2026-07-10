package schedule

import (
	"time"

	"github.com/vtak/itr-foreign-assets-helper/internal/forex"
)

// ValuationDate bundles an event date with the SBI TT-buy reference rate used
// to convert USD amounts on that date to INR. The rate carries its own date
// (the adjusted date it was taken from) as Rate.Date, so it is not duplicated
// here.
//
// A zero ValuationDate (Date.IsZero()) means "not applicable" for this record -
// e.g. the sale date of a still-held share, or the year-end date of a share
// that was already sold.
type ValuationDate struct {
	Date time.Time           // the event date (issue / sale / peak close / year end)
	Rate forex.ReferenceRate // SBI TT-buy rate applied; Rate.Date is the date it came from
}
