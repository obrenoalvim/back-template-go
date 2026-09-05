package diagnostics

import (
	"sync/atomic"

	"gorm.io/gorm"
)

// QueryCounter counts how many SQL statements a *gorm.DB executes. Used in
// tests to lock in N+1 regressions: a handler that should run a constant
// number of queries can't silently start running 1+N without a test going
// red. Not registered in production (see internal/db/db.go).
type QueryCounter struct {
	count int64
}

func (c *QueryCounter) Name() string {
	return "diagnostics:query_counter"
}

func (c *QueryCounter) Initialize(db *gorm.DB) error {
	inc := func(*gorm.DB) { atomic.AddInt64(&c.count, 1) }

	if err := db.Callback().Query().After("gorm:query").Register("diagnostics:count_query", inc); err != nil {
		return err
	}
	if err := db.Callback().Row().After("gorm:row").Register("diagnostics:count_row", inc); err != nil {
		return err
	}
	if err := db.Callback().Raw().After("gorm:raw").Register("diagnostics:count_raw", inc); err != nil {
		return err
	}
	return nil
}

func (c *QueryCounter) Count() int64 {
	return atomic.LoadInt64(&c.count)
}

func (c *QueryCounter) Reset() {
	atomic.StoreInt64(&c.count, 0)
}
