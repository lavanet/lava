package metrics

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	websocket2 "github.com/gorilla/websocket"
	"github.com/lavanet/lava/v5/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// captureSink records every event emitted through the sink so tests can
// assert on what AddMetricFor* (relay path) and sampleAndEmit (QoS path)
// hand to the OTel pipeline.
type captureSink struct {
	events    []RelayUsageEvent
	qosEvents []OptimizerQoSReportToSend
}

func (c *captureSink) Emit(event RelayUsageEvent) { c.events = append(c.events, event) }
func (c *captureSink) EmitOptimizerQoS(report OptimizerQoSReportToSend) {
	c.qosEvents = append(c.qosEvents, report)
}

func (c *captureSink) Stats() SinkStats {
	return SinkStats{Sent: uint64(len(c.events) + len(c.qosEvents))}
}
func (c *captureSink) Close() {}

type WebSocketError struct {
	ErrorReceived string `json:"Error_Received"`
}

type ErrorData struct {
	GUID   string `json:"Error_GUID"`
	Error1 string `json:"Error"`
}

func TestGetUniqueGuidResponseForError(t *testing.T) {
	plog, err := NewRPCConsumerLogs(nil, nil, nil)
	assert.Nil(t, err)

	responseError := errors.New("response error")

	errorMsg := plog.GetUniqueGuidResponseForError(responseError, "msgSeed")

	errObject := &ErrorData{}

	err = json.Unmarshal([]byte(errorMsg), errObject)
	assert.Nil(t, err)
	assert.Equal(t, errObject.GUID, "msgSeed")
	assert.Equal(t, errObject.Error1, "response error")
}

func TestGetUniqueGuidResponseDeterministic(t *testing.T) {
	plog, err := NewRPCConsumerLogs(nil, nil, nil)
	assert.Nil(t, err)

	responseError := errors.New("response error")
	utils.SetGlobalLoggingLevel("fatal") // prevent spam.
	errorMsg := plog.GetUniqueGuidResponseForError(responseError, "msgSeed")

	for i := 1; i < 10000; i++ {
		err := plog.GetUniqueGuidResponseForError(responseError, "msgSeed")

		assert.Equal(t, err, errorMsg)
	}
}

func TestAnalyzeWebSocketErrorAndWriteMessage(t *testing.T) {
	app := fiber.New()

	app.Get("/", websocket.New(func(c *websocket.Conn) {
		mt, _, _ := c.ReadMessage()
		plog, _ := NewRPCConsumerLogs(nil, nil, nil)
		responseError := errors.New("response error")
		formatterMsg := plog.AnalyzeWebSocketErrorAndGetFormattedMessage(c.LocalAddr().String(), responseError, "seed", []byte{}, "rpcType", 1*time.Millisecond)
		assert.NotNil(t, formatterMsg)
		c.WriteMessage(mt, formatterMsg)
	}))

	listenFunc := func() {
		address := "127.0.0.1:3000"
		err := app.Listen(address)
		if err != nil {
			utils.LavaFormatError("can't listen in unitests", err, utils.Attribute{Key: "address", Value: address})
		}
	}
	go listenFunc()
	defer func() {
		app.Shutdown()
	}()
	time.Sleep(time.Millisecond * 100)
	url := "ws://127.0.0.1:3000/"
	dialer := &websocket2.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Error dialing websocket connection: %s", err)
	}
	defer conn.Close()

	err = conn.WriteMessage(websocket.TextMessage, []byte("test"))
	if err != nil {
		t.Fatalf("Error writing message to websocket connection: %s", err)
	}

	_, response, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Error reading message from websocket connection: %s", err)
	}

	errObject := &WebSocketError{}

	err = json.Unmarshal(response, errObject)
	assert.Nil(t, err)

	errData := &ErrorData{}
	err = json.Unmarshal([]byte(errObject.ErrorReceived), errData)
	assert.Nil(t, err)
	assert.Equal(t, errData.GUID, "seed")
}

// TestAddMetricForHttp_OTelEventPopulated asserts that the HTTP path
// sets Success and Origin on the RelayMetrics BEFORE emitting, so the
// OTel event carries both. Passing nil for the metrics manager mirrors
// the smart-router path where SetRelayMetrics is a no-op — Success has
// to be populated by AddMetricFor* itself.
func TestAddMetricForHttp_OTelEventPopulated(t *testing.T) {
	sink := &captureSink{}
	logs, err := NewRPCConsumerLogs(nil, sink, nil)
	require.NoError(t, err)

	data := &RelayMetrics{
		Timestamp: time.Now(),
		ChainID:   "ETH",
		APIType:   "jsonrpc",
	}
	headers := map[string][]string{
		OriginHeaderKey: {"https://app.example"},
	}
	logs.AddMetricForHttp(data, nil, headers)

	require.Len(t, sink.events, 1)
	assert.True(t, sink.events[0].Success, "Success must be set before Emit")
	assert.Equal(t, "https://app.example", sink.events[0].Origin, "Origin must reach OTel")
}

func TestAddMetricForHttp_FailurePropagatesSuccessFalse(t *testing.T) {
	sink := &captureSink{}
	logs, err := NewRPCConsumerLogs(nil, sink, nil)
	require.NoError(t, err)

	data := &RelayMetrics{}
	logs.AddMetricForHttp(data, errors.New("boom"), nil)

	require.Len(t, sink.events, 1)
	assert.False(t, sink.events[0].Success)
}

// TestAddMetricForGrpc_OTelEventPopulated covers the gRPC path. The
// Origin value is read from gRPC metadata, which the AddMetricForGrpc
// path clones via strings.Clone before populating data.Origin — the
// captured event carries the value as it stood at emit time.
func TestAddMetricForGrpc_OTelEventPopulated(t *testing.T) {
	sink := &captureSink{}
	logs, err := NewRPCConsumerLogs(nil, sink, nil)
	require.NoError(t, err)

	data := &RelayMetrics{
		Timestamp: time.Now(),
		ChainID:   "LAV1",
		APIType:   "grpc",
	}
	// gRPC metadata keys are conventionally lower-cased on the wire;
	// metadata.MD.Get normalizes lookups to lowercase.
	md := metadata.Pairs(OriginHeaderKey, "https://grpc.example")
	logs.AddMetricForGrpc(data, nil, &md)

	require.Len(t, sink.events, 1)
	assert.True(t, sink.events[0].Success)
	assert.Equal(t, "https://grpc.example", sink.events[0].Origin)
}
