package chainproxy

import (
	"github.com/gofiber/fiber/v2"
	"github.com/lavanet/lava/relayer/sentry"
	"strconv"
	"time"
)

type RelayAnalytics struct {
	ProjectHash           string
	UserId                string
	Timestamp             time.Time
	UserAgent             string
	Origin                string
	XForwardedFor         string
	ChainID               string
	APIType               string
	RequestContentLength  uint64
	ResponseContentLength uint64
	ResponseStatusCode    uint
	Latency               uint // ms
	Success               bool
	Method                string
	Params                string
	ComputeUnits          uint64
	ProvidersAddresses    []string
}

func (r *RelayAnalytics) importFromSentry(sentry sentry.Sentry) {
	r.ChainID = sentry.ChainID
	r.APIType = sentry.ApiInterface
}

func (r *RelayAnalytics) importFromFiberCtx(c *fiber.Ctx) {
	r.Method = c.Method()
	reqHeaders := c.GetReqHeaders()
	r.Origin = reqHeaders["Origin"]
	r.XForwardedFor = reqHeaders["x-forwarded-for"]
	r.UserAgent = reqHeaders["User-Agent"]
	r.ProjectHash = c.Params("dappId")
	r.RequestContentLength, _ = strconv.ParseUint(reqHeaders["Content-Length"], 10, 64)
}
