package rpcconsumer

import (
	"net/http"
	"sync"
)

type CustomLavaTransport struct {
	transport          http.RoundTripper
	lock               sync.RWMutex
	secondaryTransport http.RoundTripper
}

func NewCustomLavaTransport(httpTransport http.RoundTripper, secondaryTransport http.RoundTripper) *CustomLavaTransport {
	return &CustomLavaTransport{transport: httpTransport}
}

func (c *CustomLavaTransport) SetSecondaryTransport(secondaryTransport http.RoundTripper) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.secondaryTransport = secondaryTransport
}

func (c *CustomLavaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Custom logic before the request

	// Delegate to the underlying RoundTripper (usually http.Transport)
	resp, err := c.transport.RoundTrip(req)
	// Custom logic after the request
	if err != nil {
		// If the primary transport fails, use the secondary transport
		c.lock.RLock()
		secondaryTransport := c.secondaryTransport
		c.lock.RUnlock()
		if secondaryTransport != nil {
			resp, err = secondaryTransport.RoundTrip(req)
		}
	}
	return resp, err
}
