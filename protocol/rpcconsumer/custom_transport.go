package rpcconsumer

import (
	"net/http"
)

type CustomLavaTransport struct {
	transport http.RoundTripper
}

func NewCustomLavaTransport(httpTransport http.RoundTripper) *CustomLavaTransport {
	return &CustomLavaTransport{transport: httpTransport}
}

func (c *CustomLavaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Custom logic before the request

	// Delegate to the underlying RoundTripper (usually http.Transport)
	resp, err := c.transport.RoundTrip(req)

	// Custom logic after the request
	return resp, err
}
