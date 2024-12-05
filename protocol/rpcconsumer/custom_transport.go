package rpcconsumer

import (
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/lavanet/lava/v4/utils"
)

type CustomLavaTransport struct {
	transport          http.RoundTripper
	lock               sync.RWMutex
	secondaryTransport http.RoundTripper
	consecutiveFails   atomic.Uint64 // TODO: export to metrics
}

func NewCustomLavaTransport(httpTransport http.RoundTripper, secondaryTransport http.RoundTripper) *CustomLavaTransport {
	return &CustomLavaTransport{transport: httpTransport, secondaryTransport: secondaryTransport}
}

func (c *CustomLavaTransport) SetSecondaryTransport(secondaryTransport http.RoundTripper) {
	c.lock.Lock()
	defer c.lock.Unlock()
	utils.LavaFormatDebug("Setting secondary transport for CustomLavaTransport")
	c.secondaryTransport = secondaryTransport
}

// used to switch the primary and secondary transports, in case the primary one fails too much
func (c *CustomLavaTransport) TogglePrimarySecondaryTransport() {
	c.lock.Lock()
	defer c.lock.Unlock()
	primaryTransport := c.transport
	secondaryTransport := c.secondaryTransport
	c.secondaryTransport = primaryTransport
	c.transport = secondaryTransport
}

func (c *CustomLavaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Custom logic before the request
	c.lock.RLock()
	primaryTransport := c.transport
	secondaryTransport := c.secondaryTransport
	c.lock.RUnlock()
	// Delegate to the underlying RoundTripper (usually http.Transport)
	resp, err := primaryTransport.RoundTrip(req)
	// Custom logic after the request
	if err != nil {
		c.consecutiveFails.Add(1)
		// If the primary transport fails, use the secondary transport
		if secondaryTransport != nil {
			resp, err = secondaryTransport.RoundTrip(req)
		}
	} else {
		c.consecutiveFails.Store(0)
	}
	return resp, err
}
