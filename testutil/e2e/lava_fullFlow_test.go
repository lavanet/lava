package e2e

import (
	"testing"
	"time"
)

func TestLava(t *testing.T) {
	t.Skip("skipping lava test")
	// default timeout same as `go test`
	timeout := time.Minute * 10

	if deadline, ok := t.Deadline(); ok {
		timeout = time.Until(deadline).Round(10 * time.Second)
	}

	runE2E(timeout)
}
