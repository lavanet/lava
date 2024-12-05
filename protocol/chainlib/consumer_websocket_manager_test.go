package chainlib

import (
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/lavanet/lava/v4/protocol/common"
	"github.com/stretchr/testify/assert"
)

func TestWebsocketConnectionLimiter(t *testing.T) {
	tests := []struct {
		name            string
		connectionLimit int64
		headerLimit     int64
		ipAddress       string
		forwardedIP     string
		userAgent       string
		expectSuccess   []bool
	}{
		{
			name:            "Single connection allowed",
			connectionLimit: 1,
			headerLimit:     0,
			ipAddress:       "127.0.0.1",
			forwardedIP:     "",
			userAgent:       "test-agent",
			expectSuccess:   []bool{true},
		},
		{
			name:            "Single connection allowed",
			connectionLimit: 1,
			headerLimit:     0,
			ipAddress:       "127.0.0.1",
			forwardedIP:     "",
			userAgent:       "test-agent",
			expectSuccess:   []bool{true, false},
		},
		{
			name:            "Multiple connections allowed",
			connectionLimit: 2,
			headerLimit:     0,
			ipAddress:       "127.0.0.1",
			forwardedIP:     "",
			userAgent:       "test-agent",
			expectSuccess:   []bool{true, true},
		},
		{
			name:            "Multiple connections allowed",
			connectionLimit: 2,
			headerLimit:     0,
			ipAddress:       "127.0.0.1",
			forwardedIP:     "",
			userAgent:       "test-agent",
			expectSuccess:   []bool{true, true, false},
		},
		{
			name:            "Header limit overrides global limit succeed",
			connectionLimit: 3,
			headerLimit:     2,
			ipAddress:       "127.0.0.1",
			forwardedIP:     "",
			userAgent:       "test-agent",
			expectSuccess:   []bool{true, true},
		},
		{
			name:            "Header limit overrides global limit fail",
			connectionLimit: 0,
			headerLimit:     2,
			ipAddress:       "127.0.0.1",
			forwardedIP:     "",
			userAgent:       "test-agent",
			expectSuccess:   []bool{true, true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create a new connection limiter
			wcl := &WebsocketConnectionLimiter{
				ipToNumberOfActiveConnections: make(map[string]int64),
			}

			// Set global connection limit for testing
			MaximumNumberOfParallelWebsocketConnectionsPerIp = tt.connectionLimit

			// Create mock websocket connection
			mockWsConn := NewMockWebsocketConnection(ctrl)

			// Set up expectations
			mockWsConn.EXPECT().Locals(WebSocketOpenConnectionsLimitHeader).Return(tt.headerLimit).AnyTimes()
			mockWsConn.EXPECT().Locals(common.IP_FORWARDING_HEADER_NAME).Return(tt.forwardedIP).AnyTimes()
			mockWsConn.EXPECT().Locals("User-Agent").Return(tt.userAgent).AnyTimes()
			mockWsConn.EXPECT().RemoteAddr().Return(&net.TCPAddr{
				IP:   net.ParseIP(tt.ipAddress),
				Port: 8080,
			}).AnyTimes()
			mockWsConn.EXPECT().WriteMessage(gomock.Any(), gomock.Any()).Do(func(messageType int, data []byte) {
				t.Logf("WriteMessage called with messageType: %d, data: %s", messageType, string(data))
			}).AnyTimes()

			// Test the connection
			for _, expectSuccess := range tt.expectSuccess {
				canOpen, _ := wcl.CanOpenConnection(mockWsConn)
				if expectSuccess {
					assert.True(t, canOpen, "Expected connection to be allowed")
				} else {
					assert.False(t, canOpen, "Expected connection to be denied")
				}
			}
		})
	}
}
