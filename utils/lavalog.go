package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	sdkerrors "cosmossdk.io/errors"
	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	zerolog "github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	EventPrefix = "lava_"
)

const (
	LAVA_LOG_TRACE = iota
	LAVA_LOG_DEBUG
	LAVA_LOG_INFO
	LAVA_LOG_WARN
	LAVA_LOG_ERROR
	LAVA_LOG_FATAL
	LAVA_LOG_PANIC
	LAVA_LOG_PRODUCTION
	NoColor = true
)

var (
	JsonFormat = false
	// if set to production, this will replace some errors to warning that can be caused by misuse instead of bugs
	ExtendedLogLevel      = "development"
	rollingLogLogger      = zerolog.New(os.Stderr).Level(zerolog.Disabled) // this is the singleton rolling logger.
	defaultGlobalLogLevel = zerolog.DebugLevel
)

type Attribute struct {
	Key   string
	Value interface{}
}

func StringMapToAttributes(details map[string]string) []Attribute {
	var attrs []Attribute
	for key, val := range details {
		attrs = append(attrs, Attribute{Key: key, Value: val})
	}
	return attrs
}

func LogAttr(key string, value interface{}) Attribute {
	return Attribute{Key: key, Value: value}
}

func LogLavaEvent(ctx sdk.Context, logger log.Logger, name string, attributes map[string]string, description string) {
	attributes_str := ""
	eventAttrs := []sdk.Attribute{}
	for key, val := range attributes {
		attributes_str += fmt.Sprintf("%s: %s,", key, val)
		eventAttrs = append(eventAttrs, sdk.NewAttribute(key, val))
	}
	sort.Slice(eventAttrs, func(i, j int) bool {
		return eventAttrs[i].Key < eventAttrs[j].Key
	})
	logger.Info(fmt.Sprintf("%s%s:%s %s", EventPrefix, name, description, attributes_str))
	ctx.EventManager().EmitEvent(sdk.NewEvent(EventPrefix+name, eventAttrs...))
}

func getLogLevel(logLevel string) zerolog.Level {
	switch logLevel {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

func SetGlobalLoggingLevel(logLevel string) {
	// setting global level prevents us from having two different levels for example one for stdout and one for rolling log.
	// zerolog.SetGlobalLevel(getLogLevel(logLevel))
	defaultGlobalLogLevel = getLogLevel(logLevel)
	LavaFormatInfo("setting log level", Attribute{Key: "loglevel", Value: logLevel})
}

func SetLogLevelFieldName(fieldName string) {
	zerolog.LevelFieldName = fieldName
}

func RollingLoggerSetup(rollingLogLevel string, filePath string, maxSize string, maxBackups string, maxAge string, stdFormat string) func() {
	maxSizeNumber, err := strconv.Atoi(maxSize)
	if err != nil {
		LavaFormatFatal("strconv.Atoi(maxSize)", err, LogAttr("maxSize", maxSize))
	}
	maxBackupsNumber, err := strconv.Atoi(maxBackups)
	if err != nil {
		LavaFormatFatal("strconv.Atoi(maxSize)", err, LogAttr("maxBackups", maxBackups))
	}
	maxAgeNumber, err := strconv.Atoi(maxAge)
	if err != nil {
		LavaFormatFatal("strconv.Atoi(maxSize)", err, LogAttr("maxAge", maxAge))
	}

	rollingLogOutput := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    maxSizeNumber,
		MaxBackups: maxBackupsNumber,
		MaxAge:     maxAgeNumber,
		Compress:   true,
	}
	var logLevel zerolog.Level
	switch rollingLogLevel {
	case "off":
		return func() {} // default is disabled.
	case "trace":
		logLevel = zerolog.TraceLevel
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	case "fatal":
		logLevel = zerolog.FatalLevel
	default:
		LavaFormatFatal("unsupported case for rollingLoggerSetup", nil, LogAttr("rollingLogLevel", rollingLogLevel))
	}
	// set the rolling log level.
	if stdFormat == "json" {
		rollingLogLogger = zerolog.New(rollingLogOutput).Level(logLevel).With().Timestamp().Logger()
	} else {
		rollingLogLogger = zerolog.New(zerolog.ConsoleWriter{Out: rollingLogOutput, NoColor: NoColor, TimeFormat: time.Stamp}).Level(logLevel).With().Timestamp().Logger()
	}
	rollingLogLogger.Debug().Msg("Starting Rolling Logger")
	return func() { rollingLogOutput.Close() }
}

func StrValueForLog(val interface{}, key string, idx int, attributes []Attribute) string {
	st_val := ""
	switch value := val.(type) {
	case context.Context:
		// we don't want to print the whole context so change it
		switch key {
		case "GUID":
			guid, found := GetUniqueIdentifier(value)
			if found {
				st_val = strconv.FormatUint(guid, 10)
				attributes[idx] = Attribute{Key: key, Value: guid}
			} else {
				attributes[idx] = Attribute{Key: key, Value: "no-guid"}
			}
		default:
			attributes[idx] = Attribute{Key: key, Value: "context-masked"}
		}
	default:
		st_val = StrValue(val)
	}
	return st_val
}

func StrValue(val interface{}) string {
	st_val := ""
	switch value := val.(type) {
	case context.Context:
		// we don't want to print the whole context so change it
	case bool:
		if value {
			st_val = "true"
		} else {
			st_val = "false"
		}
	case fmt.Stringer:
		st_val = value.String()
	case string:
		st_val = value
	case int:
		st_val = strconv.Itoa(value)
	case int64:
		st_val = strconv.FormatInt(value, 10)
	case uint64:
		st_val = strconv.FormatUint(value, 10)
	case uint32:
		st_val = strconv.FormatUint(uint64(value), 10)
	case error:
		st_val = value.Error()
	case []error:
		for _, err := range value {
			if err == nil {
				continue
			}
			st_val += err.Error() + ";"
		}
	case []string:
		st_val = strings.Join(value, ",")
	// needs to come after stringer so byte inheriting objects will use their string method if implemented (like AccAddress)
	case []byte:
		st_val = string(value)
	case nil:
		st_val = ""
	default:
		st_val = fmt.Sprintf("%+v", value)
	}
	return st_val
}

// ExtractErrorStructure extracts structured information from errors for ELK-friendly logging.
// It uses a generic pattern-based approach to handle various error types.
func ExtractErrorStructure(err error) map[string]interface{} {
	if err == nil {
		return nil
	}

	result := make(map[string]interface{})
	errMsg := err.Error()

	// Step 1: Check if it's an sdkerrors.Error (only works for direct, non-gRPC errors)
	var sdkErr *sdkerrors.Error
	if errors.As(err, &sdkErr) {
		result["error_code"] = sdkErr.ABCICode()
		result["error_codespace"] = sdkErr.Codespace()
		result["error"] = errMsg
		result["error_description"] = parseErrorDescription(errMsg)
		return result
	}

	// Step 2: Check if it's a gRPC status error (most common case after network hop)
	if st, ok := status.FromError(err); ok {
		code := st.Code()
		msg := st.Message()

		// Extract error code - try multiple sources
		if errorCode := extractErrorCode(code, msg); errorCode > 0 {
			result["error_code"] = errorCode
		} else if uint32(code) <= 16 {
			// Standard gRPC codes: use meaningful name
			result["grpc_code"] = code.String()
		}

		result["error"] = msg
		result["error_description"] = parseErrorDescription(msg)

		// Extract additional details if available
		if details := parseErrorDetails(msg); details != "" {
			result["error_details"] = details
		}

		return result
	}

	// Step 3: Fallback for plain errors - parse generically
	result["error"] = errMsg
	result["error_description"] = parseErrorDescription(errMsg)

	if details := parseErrorDetails(errMsg); details != "" {
		result["error_details"] = details
	}

	return result
}

// extractErrorCode attempts to extract a numeric error code from various sources
func extractErrorCode(grpcCode codes.Code, message string) uint32 {
	// First, check if gRPC code itself is a custom code (>16)
	if uint32(grpcCode) > 16 {
		return uint32(grpcCode)
	}

	// Try to find embedded code in message: "Code(3369)" or "code = Code(3369)"
	re := regexp.MustCompile(`[Cc]ode\((\d+)\)`)
	if matches := re.FindStringSubmatch(message); len(matches) > 1 {
		if code, err := strconv.ParseUint(matches[1], 10, 32); err == nil {
			return uint32(code)
		}
	}

	return 0
}

// parseErrorDescription extracts a high-level description from error message
// Generic pattern: takes text before first colon or "ErrMsg"
func parseErrorDescription(message string) string {
	// Remove common prefixes
	msg := strings.TrimPrefix(message, "rpc error: ")
	msg = strings.TrimPrefix(msg, "code = ")

	// Pattern 1: "desc = <description>: ..." or "desc = <description>"
	if idx := strings.Index(msg, "desc = "); idx >= 0 {
		msg = msg[idx+7:] // Skip "desc = "
	}

	// Pattern 2: Take text before first colon (description: details)
	if idx := strings.Index(msg, ":"); idx > 0 {
		return strings.TrimSpace(msg[:idx])
	}

	// Pattern 3: Take text before "ErrMsg"
	if idx := strings.Index(msg, " ErrMsg"); idx > 0 {
		return strings.TrimSpace(msg[:idx])
	}

	// No clear pattern - return first 100 chars as description
	if len(msg) > 100 {
		return strings.TrimSpace(msg[:100])
	}

	return strings.TrimSpace(msg)
}

// parseErrorDetails extracts detailed information from error message
// Generic pattern: takes text after first colon
func parseErrorDetails(message string) string {
	// Skip common prefixes
	msg := strings.TrimPrefix(message, "rpc error: ")
	msg = strings.TrimPrefix(msg, "code = ")
	if idx := strings.Index(msg, "desc = "); idx >= 0 {
		msg = msg[idx+7:]
	}

	// Pattern 1: Text after "ErrMsg: "
	if idx := strings.Index(msg, "ErrMsg: "); idx >= 0 {
		details := msg[idx+8:]
		// Remove trailing error chain parts
		if endIdx := strings.Index(details, "{"); endIdx > 0 {
			details = details[:endIdx]
		}
		return strings.TrimSpace(details)
	}

	// Pattern 2: Text after first colon
	if idx := strings.Index(msg, ":"); idx > 0 && idx < len(msg)-1 {
		details := msg[idx+1:]
		// Take only the first segment (before next colon)
		if nextIdx := strings.Index(details, ":"); nextIdx > 0 {
			details = details[:nextIdx]
		}
		return strings.TrimSpace(details)
	}

	return ""
}

func LavaFormatLog(description string, err error, attributes []Attribute, severity uint) error {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if JsonFormat {
		zerologlog.Logger = zerologlog.Output(os.Stderr).Level(defaultGlobalLogLevel)
	} else {
		zerologlog.Logger = zerologlog.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: NoColor, TimeFormat: time.Stamp}).Level(defaultGlobalLogLevel)
	}

	// depending on the build flag, this log function will log either a warning or an error.
	// the purpose of this function is to fail E2E tests and not allow unexpected behavior to reach main.
	// while in production some errors may occur as consumers / providers might set up their processes in the wrong way.
	// in test environment we don't expect to have these errors and if they occur we would like to fail the test.
	if severity == LAVA_LOG_PRODUCTION {
		if ExtendedLogLevel == "production" {
			severity = LAVA_LOG_WARN
		} else {
			severity = LAVA_LOG_ERROR
		}
	}

	var logEvent *zerolog.Event
	var rollingLoggerEvent *zerolog.Event
	switch severity {
	case LAVA_LOG_PANIC:
		// prefix = "Panic:"
		logEvent = zerologlog.Panic()
		if rollingLogLogger.GetLevel() != zerolog.Disabled {
			rollingLoggerEvent = rollingLogLogger.Panic()
		}
	case LAVA_LOG_FATAL:
		// prefix = "Fatal:"
		logEvent = zerologlog.Fatal()
		if rollingLogLogger.GetLevel() != zerolog.Disabled {
			rollingLoggerEvent = rollingLogLogger.Fatal()
		}
	case LAVA_LOG_ERROR:
		// prefix = "Error:"
		logEvent = zerologlog.Error()
		rollingLoggerEvent = rollingLogLogger.Error()
	case LAVA_LOG_WARN:
		// prefix = "Warning:"
		logEvent = zerologlog.Warn()
		rollingLoggerEvent = rollingLogLogger.Warn()
	case LAVA_LOG_INFO:
		logEvent = zerologlog.Info()
		rollingLoggerEvent = rollingLogLogger.Info()
		// prefix = "Info:"
	case LAVA_LOG_DEBUG:
		logEvent = zerologlog.Debug()
		rollingLoggerEvent = rollingLogLogger.Debug()
		// prefix = "Debug:"
	case LAVA_LOG_TRACE:
		logEvent = zerologlog.Trace()
		rollingLoggerEvent = rollingLogLogger.Trace()
		// prefix = "Trace:"
	}

	// Handle error structurally - extract fields instead of concatenating strings
	if err != nil {
		structuredErr := ExtractErrorStructure(err)
		if len(structuredErr) > 0 {
			// Add each extracted error field as a separate structured field
			for key, val := range structuredErr {
				strVal := StrValue(val)
				logEvent = logEvent.Str(key, strVal)
				rollingLoggerEvent = rollingLoggerEvent.Str(key, strVal)
			}
		} else {
			// Fallback: use standard error field if extraction fails
			logEvent = logEvent.Err(err)
			rollingLoggerEvent = rollingLoggerEvent.Err(err)
		}
	}

	// Add attributes as structured fields (NO string concatenation)
	if len(attributes) > 0 {
		for idx, attr := range attributes {
			key := attr.Key
			val := attr.Value
			st_val := StrValueForLog(val, key, idx, attributes)
			logEvent = logEvent.Str(key, st_val)
			rollingLoggerEvent = rollingLoggerEvent.Str(key, st_val)
		}
	}

	// Emit clean message: ONLY the description, no concatenated error or attributes
	logEvent.Msg(description)
	rollingLoggerEvent.Msg(description)

	// Return wrapped error for backward compatibility with calling code
	// The error object still contains full context for error handling
	errRet := sdkerrors.Wrap(err, description)
	if errRet == nil { // we always want to return an error if lavaFormatError was called
		return fmt.Errorf("%s", description)
	}
	return errRet
}

func LavaFormatPanic(description string, err error, attributes ...Attribute) {
	attributes = append(attributes, Attribute{Key: "StackTrace", Value: debug.Stack()})
	LavaFormatLog(description, err, attributes, LAVA_LOG_PANIC)
}

func LavaFormatFatal(description string, err error, attributes ...Attribute) {
	attributes = append(attributes, Attribute{Key: "StackTrace", Value: debug.Stack()})
	LavaFormatLog(description, err, attributes, LAVA_LOG_FATAL)
}

// see documentation in LavaFormatLog function
func LavaFormatProduction(description string, err error, attributes ...Attribute) error {
	return LavaFormatLog(description, err, attributes, LAVA_LOG_PRODUCTION)
}

func LavaFormatError(description string, err error, attributes ...Attribute) error {
	return LavaFormatLog(description, err, attributes, LAVA_LOG_ERROR)
}

func LavaFormatWarning(description string, err error, attributes ...Attribute) error {
	return LavaFormatLog(description, err, attributes, LAVA_LOG_WARN)
}

func LavaFormatInfo(description string, attributes ...Attribute) error {
	return LavaFormatLog(description, nil, attributes, LAVA_LOG_INFO)
}

func LavaFormatDebug(description string, attributes ...Attribute) error {
	return LavaFormatLog(description, nil, attributes, LAVA_LOG_DEBUG)
}

func LavaFormatTrace(description string, attributes ...Attribute) error {
	return LavaFormatLog(description, nil, attributes, LAVA_LOG_TRACE)
}

func IsTraceLogLevelEnabled() bool {
	return defaultGlobalLogLevel == zerolog.TraceLevel
}

func FormatStringerList[T fmt.Stringer](description string, listToPrint []T, separator string) string {
	st := ""
	for _, printable := range listToPrint {
		st = st + separator + printable.String() + "\n"
	}
	st = fmt.Sprintf(description+"\n\t%s", st)
	return st
}

func FormatLongString(msg string, maxCharacters int) string {
	if maxCharacters != 0 && len(msg) > maxCharacters {
		postfixLen := maxCharacters / 3
		prefixLen := maxCharacters - postfixLen
		return msg[:prefixLen] + "...truncated..." + msg[len(msg)-postfixLen:]
	}
	return msg
}

func ToHexString(hash string) string {
	return fmt.Sprintf("%x", hash)
}
