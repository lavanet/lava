package relaycore

import (
	"errors"
	"testing"
)

var errSanitizeBench = errors.New("relay failed: provider timeout SessionId:1867231 GUID:9817236406184312 response code 504 from upstream")

func BenchmarkSanitizeError(b *testing.B) {
	r := &RelayErrors{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = r.sanitizeError(errSanitizeBench)
	}
}
