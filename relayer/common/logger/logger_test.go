package logger

import (
	zerologlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestGetInstance tests that GetInstance returns a singleton instance
func TestGetInstance(t *testing.T) {
	logger1 := GetInstance()
	logger2 := GetInstance()
	assert.Equal(t, logger1, logger2)
}

// TestLog tests that Log function pushes log messages to the log channel
func TestLog(t *testing.T) {
	logger := GetInstance()
	log := LogMessage{Description: "test log", Err: nil, LogEvent: zerologlog.Info()}
	logger.Log(log)
	select {
	case msg := <-logger.logChan:
		assert.Equal(t, "test log", msg.Description) // test Description
		assert.Equal(t, log.LogEvent, msg.LogEvent)  // test log level
		assert.Nil(t, msg.Err)                       // test error
	default:
		t.Error("log message not received") // if message not received then throws error
	}
}
