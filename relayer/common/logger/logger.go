package logger

import (
	"sync"

	"github.com/rs/zerolog"
)

type LogMessage struct {
	Description string         // a string describing the log message
	Err         error          // an error associated with the log message
	LogEvent    *zerolog.Event // log level
}

var (
	instance *Logger
	once     sync.Once
)

type Logger struct {
	logChan chan LogMessage // channel to send log messages
}

// GetInstance is a function that creates a singleton instance of the Logger struct
// and returns it.
func GetInstance() *Logger {
	once.Do(func() {
		instance = &Logger{
			logChan: make(chan LogMessage, 1024), // the channel buffer size is 1024
		}
		go instance.listen()
	})
	return instance
}

// listen function to listen on the log channel, it will keep running and waiting for new messages
// on the channel and it will print the logs once it receives any log message
func (l *Logger) listen() {
	for {
		// wait for a message
		msg := <-l.logChan

		// log the message
		l.printLogs(msg.Description, msg.LogEvent)
	}
}

// printLogs function is to print the log messages
func (l *Logger) printLogs(description string, logEvent *zerolog.Event) {
	logEvent.Msg(description)
}

// Log function is to push the log messages to the channel
func (l *Logger) Log(msg LogMessage) {
	l.logChan <- msg
}
