package common

import (
	"net/http"
	"testing"
	"time"
)

// TestOptimizedHttpTransport verifies that the optimized HTTP transport
// is configured with the correct values for high-concurrency scenarios
func TestOptimizedHttpTransport(t *testing.T) {
	transport := OptimizedHttpTransport()

	if transport == nil {
		t.Fatal("OptimizedHttpTransport returned nil")
	}

	// Verify connection pool settings
	if transport.MaxIdleConns != DefaultMaxIdleConns {
		t.Errorf("MaxIdleConns = %d, want %d", transport.MaxIdleConns, DefaultMaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != DefaultMaxIdleConnsPerHost {
		t.Errorf("MaxIdleConnsPerHost = %d, want %d (critical for blockchain node connections)",
			transport.MaxIdleConnsPerHost, DefaultMaxIdleConnsPerHost)
	}

	if transport.MaxConnsPerHost != DefaultMaxConnsPerHost {
		t.Errorf("MaxConnsPerHost = %d, want %d", transport.MaxConnsPerHost, DefaultMaxConnsPerHost)
	}

	// Verify timeout settings
	if transport.IdleConnTimeout != DefaultIdleConnTimeout {
		t.Errorf("IdleConnTimeout = %v, want %v", transport.IdleConnTimeout, DefaultIdleConnTimeout)
	}

	if transport.TLSHandshakeTimeout != DefaultTLSHandshakeTimeout {
		t.Errorf("TLSHandshakeTimeout = %v, want %v", transport.TLSHandshakeTimeout, DefaultTLSHandshakeTimeout)
	}

	if transport.ExpectContinueTimeout != DefaultExpectContinueTimeout {
		t.Errorf("ExpectContinueTimeout = %v, want %v", transport.ExpectContinueTimeout, DefaultExpectContinueTimeout)
	}

	if transport.ResponseHeaderTimeout != DefaultResponseHeaderTimeout {
		t.Errorf("ResponseHeaderTimeout = %v, want %v", transport.ResponseHeaderTimeout, DefaultResponseHeaderTimeout)
	}

	// Verify HTTP/2 is enabled
	if !transport.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 should be true for better performance")
	}

	// Verify proxy configuration
	if transport.Proxy == nil {
		t.Error("Proxy should be set to http.ProxyFromEnvironment")
	}

	// Verify DialContext is configured
	if transport.DialContext == nil {
		t.Error("DialContext should be configured with dialer settings")
	}
}

// TestOptimizedHttpClient verifies that the optimized HTTP client
// is properly configured with the optimized transport and timeout
func TestOptimizedHttpClient(t *testing.T) {
	client := OptimizedHttpClient()

	if client == nil {
		t.Fatal("OptimizedHttpClient returned nil")
	}

	// Verify client timeout
	if client.Timeout != DefaultHTTPTimeout {
		t.Errorf("Client Timeout = %v, want %v (needed for slow blockchain operations like trace_block)",
			client.Timeout, DefaultHTTPTimeout)
	}

	// Verify transport is set
	if client.Transport == nil {
		t.Fatal("Client Transport should not be nil")
	}

	// Verify transport is the correct type
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Client Transport is not *http.Transport, got %T", client.Transport)
	}

	// Verify transport has optimized settings (spot check key values)
	if transport.MaxIdleConnsPerHost != DefaultMaxIdleConnsPerHost {
		t.Errorf("Transport MaxIdleConnsPerHost = %d, want %d",
			transport.MaxIdleConnsPerHost, DefaultMaxIdleConnsPerHost)
	}

	if transport.MaxIdleConns != DefaultMaxIdleConns {
		t.Errorf("Transport MaxIdleConns = %d, want %d",
			transport.MaxIdleConns, DefaultMaxIdleConns)
	}
}

// TestDefaultConstants verifies that the default constants are set to expected values
// This test documents the expected configuration and will fail if values are changed
func TestDefaultConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      interface{}
		want     interface{}
		critical bool // Mark if this value is critical for preventing connection exhaustion
	}{
		{
			name:     "MaxIdleConns",
			got:      DefaultMaxIdleConns,
			want:     200,
			critical: false,
		},
		{
			name:     "MaxIdleConnsPerHost",
			got:      DefaultMaxIdleConnsPerHost,
			want:     50,
			critical: true, // Critical: Go's default is only 2!
		},
		{
			name:     "MaxConnsPerHost",
			got:      DefaultMaxConnsPerHost,
			want:     0,
			critical: false,
		},
		{
			name:     "IdleConnTimeout",
			got:      DefaultIdleConnTimeout,
			want:     90 * time.Second,
			critical: false,
		},
		{
			name:     "TLSHandshakeTimeout",
			got:      DefaultTLSHandshakeTimeout,
			want:     10 * time.Second,
			critical: false,
		},
		{
			name:     "ExpectContinueTimeout",
			got:      DefaultExpectContinueTimeout,
			want:     1 * time.Second,
			critical: false,
		},
		{
			name:     "ResponseHeaderTimeout",
			got:      int64(DefaultResponseHeaderTimeout),
			want:     int64(0),
			critical: false,
		},
		{
			name:     "DialTimeout",
			got:      DefaultDialTimeout,
			want:     10 * time.Second,
			critical: false,
		},
		{
			name:     "KeepAlive",
			got:      DefaultKeepAlive,
			want:     30 * time.Second,
			critical: false,
		},
		{
			name:     "HTTPTimeout",
			got:      DefaultHTTPTimeout,
			want:     5 * time.Minute,
			critical: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				severity := "ERROR"
				if tt.critical {
					severity = "CRITICAL"
				}
				t.Errorf("[%s] %s = %v, want %v", severity, tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestOptimizedTransportImprovesOverDefaults verifies that our optimized settings
// are better than Go's defaults for high-concurrency blockchain node scenarios
func TestOptimizedTransportImprovesOverDefaults(t *testing.T) {
	optimized := OptimizedHttpTransport()
	defaultTransport := &http.Transport{}

	// MaxIdleConnsPerHost: Most critical improvement
	// Go default is 2, which causes 200 requests = 200 TCP connections
	// Our setting of 50 allows connection reuse
	if optimized.MaxIdleConnsPerHost <= defaultTransport.MaxIdleConnsPerHost {
		t.Errorf("Optimized MaxIdleConnsPerHost (%d) should be greater than default (%d) to prevent connection exhaustion",
			optimized.MaxIdleConnsPerHost, defaultTransport.MaxIdleConnsPerHost)
	}

	// Verify our value is significantly higher (at least 10x improvement)
	if optimized.MaxIdleConnsPerHost < 20 {
		t.Errorf("MaxIdleConnsPerHost (%d) seems too low for high-concurrency scenarios (200+ requests)",
			optimized.MaxIdleConnsPerHost)
	}
}

// TestMultipleClientsShareTransportSettings verifies that creating multiple
// clients produces independent instances with the same configuration
func TestMultipleClientsShareTransportSettings(t *testing.T) {
	client1 := OptimizedHttpClient()
	client2 := OptimizedHttpClient()

	if client1 == client2 {
		t.Error("OptimizedHttpClient should create new client instances, not return the same pointer")
	}

	// Verify both have the same timeout
	if client1.Timeout != client2.Timeout {
		t.Errorf("Client timeouts differ: client1=%v, client2=%v", client1.Timeout, client2.Timeout)
	}

	// Verify transports are also independent instances
	transport1, ok := client1.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Client1 Transport is not *http.Transport, got %T", client1.Transport)
	}
	transport2, ok := client2.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Client2 Transport is not *http.Transport, got %T", client2.Transport)
	}

	if transport1 == transport2 {
		t.Error("OptimizedHttpClient should create new transport instances")
	}

	// But they should have the same configuration
	if transport1.MaxIdleConnsPerHost != transport2.MaxIdleConnsPerHost {
		t.Error("Transport configurations should match across client instances")
	}
}

// TestOptimizedTransportDialerSettings verifies that the dialer is properly configured
func TestOptimizedTransportDialerSettings(t *testing.T) {
	transport := OptimizedHttpTransport()

	// Create a test dialer to verify settings are applied correctly
	// We can't directly inspect the DialContext closure, but we can verify
	// the transport has it configured
	if transport.DialContext == nil {
		t.Fatal("DialContext should be configured")
	}

	// Verify the dialer timeout is reasonable
	// The DialContext uses DefaultDialTimeout which should be set
	expectedDialTimeout := DefaultDialTimeout
	if expectedDialTimeout == 0 {
		t.Error("DefaultDialTimeout should not be 0 - connections could hang indefinitely")
	}

	if expectedDialTimeout > 30*time.Second {
		t.Errorf("DefaultDialTimeout (%v) seems too long for responsive blockchain queries", expectedDialTimeout)
	}
}

// BenchmarkOptimizedHttpTransportCreation benchmarks the cost of creating the transport
func BenchmarkOptimizedHttpTransportCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = OptimizedHttpTransport()
	}
}

// BenchmarkOptimizedHttpClientCreation benchmarks the cost of creating the client
func BenchmarkOptimizedHttpClientCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = OptimizedHttpClient()
	}
}
