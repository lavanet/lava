package chainproxy

import (
	"github.com/gofiber/fiber/v2"
	"github.com/lavanet/lava/relayer/sentry"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"strconv"
	"time"
)

type RelayAnalytics struct {
	ProjectHash   string // imported - REST
	Timestamp     time.Time
	UserAgent     string // imported - REST
	Origin        string // imported - REST
	XForwardedFor string // imported - REST

	RequestMethod         string /// imported - REST
	RequestContentLength  uint64 /// imported - REST
	ResponseContentLength int    //  Done - REST
	ResponseStatusCode    uint
	Method                string //
	Params                string //
	ChainID               string /// DONE - SendRelay
	APIType               string /// DONE - SendRelay

	// latency in ms
	Latency         int64  // DONE - SendRelay
	Success         bool   // Done - SendRelay
	ComputeUnits    uint64 // Done - SendRelay
	ProviderAddress string // Done - SendRelay
}

func NewRelayAnalytics() *RelayAnalytics {
	return &RelayAnalytics{Timestamp: time.Now()}
}

func (r *RelayAnalytics) importFromSentry(sentry *sentry.Sentry) {
	r.ChainID = sentry.ChainID
	r.APIType = sentry.ApiInterface
}

func (r *RelayAnalytics) importFromFiberCtx(c *fiber.Ctx) {
	r.RequestMethod = c.Method()
	reqHeaders := c.GetReqHeaders()
	r.Origin = reqHeaders["Origin"]
	r.XForwardedFor = reqHeaders["x-forwarded-for"]
	r.UserAgent = reqHeaders["User-Agent"]
	r.ProjectHash = c.Params("dappId")
	r.RequestContentLength, _ = strconv.ParseUint(reqHeaders["Content-Length"], 10, 64)
}

func (r *RelayAnalytics) importFromRelayRequest(req *pairingtypes.RelayRequest) {
	r.ChainID = req.ChainID
	r.ComputeUnits = req.CuSum
	r.ProviderAddress = req.Provider
}
