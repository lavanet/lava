package metrics

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	websocket2 "github.com/gorilla/websocket"
	"github.com/lavanet/lava/utils"
	"github.com/stretchr/testify/assert"
)

type WebSocketError struct {
	ErrorReceived string `json:"Error_Received"`
}

type ErrorData struct {
	GUID   string `json:"Error_GUID"`
	Error1 string `json:"Error"`
}

func TestGetUniqueGuidResponseForError(t *testing.T) {
	plog, err := NewRPCConsumerLogs(nil, nil)
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
	plog, err := NewRPCConsumerLogs(nil, nil)
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
		plog, _ := NewRPCConsumerLogs(nil, nil)
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
