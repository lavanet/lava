package utils

import (
	"strconv"
)

// this logger is used for any third party loggers that require the logging functionality
// to be a bit different than the lava logger functionality
type LoggerWrapper struct {
	LoggerName string
}

func (lw LoggerWrapper) getAttributes(extraInfo ...interface{}) []Attribute {
	attributes := make([]Attribute, len(extraInfo))
	for idx, info := range extraInfo {
		attributes[idx] = Attribute{Key: strconv.Itoa(idx), Value: info}
	}
	return attributes
}

func (lw LoggerWrapper) Errorf(msg string, extraInfo ...interface{}) {
	LavaFormatError(lw.LoggerName+msg, nil, lw.getAttributes(extraInfo)...)
}

func (lw LoggerWrapper) Warningf(msg string, extraInfo ...interface{}) {
	LavaFormatWarning(lw.LoggerName+msg, nil, lw.getAttributes(extraInfo)...)
}

func (lw LoggerWrapper) Infof(msg string, extraInfo ...interface{}) {
	// Check level before constructing attributes to avoid allocations
	if !IsInfoLevelEnabled() {
		return
	}
	LavaFormatInfo(lw.LoggerName+msg, lw.getAttributes(extraInfo)...)
}

func (lw LoggerWrapper) Debugf(msg string, extraInfo ...interface{}) {
	// Check level before constructing attributes to avoid allocations
	if !IsDebugLevelEnabled() {
		return
	}
	LavaFormatDebug(lw.LoggerName+msg, lw.getAttributes(extraInfo)...)
}
