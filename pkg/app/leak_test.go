package app

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain verifies that no goroutine started by tests in this package
// outlives the test binary. Catches regressions like missing
// WorkerService.Stop or unbounded background tickers.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
		goleak.IgnoreTopFunction("net/http.(*Transport).dialConnFor"),
	)
}
