package logger

import (
	"fmt"
	"sync"
	"testing"
	"time"

	zerologlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Test_isInsideEpochErrors tests if the error message is inside epoch errors
func Test_isInsideEpochErrors(t *testing.T) {
	logger := GetInstance()
	logger.ResetErrorAllowList()

	require.True(t, logger.isInsideEpochErrors(NoPairingAvailableError))
	require.False(t, logger.isInsideEpochErrors(15))
}

// TestStressLogger tests that we never reach max capacity in the logger channel
func TestStressLogger(t *testing.T) {
	// create a wait group
	var wg sync.WaitGroup

	// get the logger instance
	l := GetInstance()

	// start 100 goroutines
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				// check if the channel is full
				if len(l.logChan) == cap(l.logChan) {
					t.Errorf("goroutine %d: channel is full", i)
				}
				// send a log message
				l.Log(LogMessage{
					Description: fmt.Sprintf("log message from goroutine %d", i),
					LogEvent:    zerologlog.Info(),
				})

				time.Sleep(3 * time.Millisecond)
			}
		}(i)
	}

	// wait for all goroutines to complete
	wg.Wait()
}
