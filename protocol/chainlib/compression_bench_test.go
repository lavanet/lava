package chainlib

import (
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/require"
)

// LAVA_BENCH_HUGE=1 opts into the huge_* payload buckets (128 MB / 1 GB).
// They are skipped by default because a single 1 GB brotli iteration takes
// ~15–20 s on a single core, and allocating 1 GB on CI is unfriendly.
const hugeBenchEnvVar = "LAVA_BENCH_HUGE"

// compressionBenchSizes defines the payload buckets used by both the benchmark
// and the companion ratio test. JSON-RPC traffic on ETH routers clusters around
// these shapes in practice:
//
//	small  — eth_blockNumber / eth_chainId
//	medium — single eth_getTransactionReceipt, moderate eth_call
//	large  — eth_getLogs over a multi-block range, debug_trace*
var compressionBenchSizes = []struct {
	name  string
	bytes int
	huge  bool // gated behind LAVA_BENCH_HUGE=1
}{
	{"small_~256B", 256, false},
	{"medium_~16KB", 16 * 1024, false},
	{"large_~1MB", 1024 * 1024, false},
	{"huge_~128MB", 128 * 1024 * 1024, true},
	{"huge_~1GB", 1024 * 1024 * 1024, true},
}

var compressionBenchModes = []string{
	common.ResponseCompressionOff,
	common.ResponseCompressionGzip,
	common.ResponseCompressionBrotli,
}

// BenchmarkResponseCompression measures per-request CPU cost of the three
// response-compression modes across small / medium / large JSON-RPC-shaped
// payloads. It exercises the full fiber middleware chain so numbers reflect
// what production actually pays, not just the raw encoder.
//
//	go test ./protocol/chainlib/ -bench ResponseCompression -benchmem -run=^$ -count=3
//
// MB/s (via SetBytes) is the metric to compare: higher = less CPU per byte of
// response. Wire-size savings are covered by TestCompressionRatios below.
func BenchmarkResponseCompression(b *testing.B) {
	hugeEnabled := os.Getenv(hugeBenchEnvVar) != ""
	for _, size := range compressionBenchSizes {
		if size.huge && !hugeEnabled {
			continue
		}
		payload := generateJSONRPCPayload(size.bytes)
		for _, mode := range compressionBenchModes {
			b.Run(fmt.Sprintf("%s/%s", size.name, mode), func(b *testing.B) {
				app := fiber.New(fiber.Config{
					DisableStartupMessage: true,
					// Allow 1 GB bodies through fasthttp (default is 4 MB).
					BodyLimit: 2 * 1024 * 1024 * 1024,
				})
				applyResponseCompression(app, mode)
				app.Get("/", func(c *fiber.Ctx) error {
					c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
					return c.Send(payload)
				})

				b.SetBytes(int64(len(payload)))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					req := httptest.NewRequest(fiber.MethodGet, "/", nil)
					req.Header.Set(fiber.HeaderAcceptEncoding, "br, gzip, deflate")
					resp, err := app.Test(req, -1)
					if err != nil {
						b.Fatalf("request %d: %v", i, err)
					}
					// Drain + close so fasthttp recycles the response buffer.
					_, _ = io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			})
		}
	}
}

// TestCompressionRatios prints the wire-size savings of each mode vs uncompressed
// for each payload bucket. Running it alongside the benchmark gives a full
// picture: benchmark = CPU cost per byte, this test = bandwidth saved per byte.
//
// Not an assertion test — it always passes. Run with -v to see the table:
//
//	go test ./protocol/chainlib/ -run TestCompressionRatios -v
//
// Ratios in this table are upper bounds. The generated payload has repeating
// structure (zero-padded hex fields) that compresses better than real-world
// JSON-RPC responses would. Expect production ratios closer to 3–5× for
// typical eth_getLogs responses, not the 25–30× shown here. CPU throughput
// numbers (from the benchmark) are load-bearing regardless, because encoder
// CPU is dominated by traversal length, not entropy.
func TestCompressionRatios(t *testing.T) {
	hugeEnabled := os.Getenv(hugeBenchEnvVar) != ""

	t.Logf("%-14s %-8s %14s %14s %8s %8s",
		"size", "mode", "original_B", "encoded_B", "ratio", "saved_%")

	for _, size := range compressionBenchSizes {
		if size.huge && !hugeEnabled {
			continue
		}
		payload := generateJSONRPCPayload(size.bytes)
		for _, mode := range compressionBenchModes {
			app := fiber.New(fiber.Config{
				DisableStartupMessage: true,
				BodyLimit:             2 * 1024 * 1024 * 1024,
			})
			applyResponseCompression(app, mode)
			app.Get("/", func(c *fiber.Ctx) error {
				c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
				return c.Send(payload)
			})

			req := httptest.NewRequest(fiber.MethodGet, "/", nil)
			req.Header.Set(fiber.HeaderAcceptEncoding, "br, gzip, deflate")
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			encoded, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			require.NoError(t, err)

			ratio := float64(len(payload)) / float64(len(encoded))
			savedPct := 100.0 * (1.0 - float64(len(encoded))/float64(len(payload)))
			t.Logf("%-14s %-8s %14d %14d %7.2fx %7.1f%%",
				size.name, mode, len(payload), len(encoded), ratio, savedPct)
		}
	}
}

// generateJSONRPCPayload returns a deterministic JSON-RPC-shaped payload of
// approximately targetBytes. The shape mimics eth_getLogs: an array of log
// entries with hex addresses, topics, and data fields. Realistic JSON matters
// because compression ratios collapse on entropic input, which would make the
// benchmark misleading.
func generateJSONRPCPayload(targetBytes int) []byte {
	// Small bucket: a single eth_chainId / eth_blockNumber–shaped payload.
	if targetBytes <= 512 {
		pad := targetBytes - 36
		if pad < 0 {
			pad = 0
		}
		return []byte(`{"jsonrpc":"2.0","id":1,"result":"0x` + strings.Repeat("a", pad) + `"}`)
	}

	const entryTemplate = `{"address":"0x%040d","topics":["0x%064d","0x%064d"],"data":"0x%0512d","blockNumber":"0x%x","transactionHash":"0x%064d","logIndex":"0x%x","removed":false}`

	var buf strings.Builder
	buf.Grow(targetBytes + 512)
	buf.WriteString(`{"jsonrpc":"2.0","id":1,"result":[`)
	for i := 0; buf.Len() < targetBytes; i++ {
		if i > 0 {
			buf.WriteByte(',')
		}
		fmt.Fprintf(&buf, entryTemplate, i, i+1, i+2, i+3, i+1_000_000, i+4, i)
	}
	buf.WriteString(`]}`)
	return []byte(buf.String())
}
