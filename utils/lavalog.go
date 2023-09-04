package utils

import (
	"context"
	"fmt"
	"os"
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
)

const (
	EventPrefix = "lava_"
)

const (
	LAVA_LOG_DEBUG = iota
	LAVA_LOG_INFO
	LAVA_LOG_WARN
	LAVA_LOG_ERROR
	LAVA_LOG_FATAL
	LAVA_LOG_PANIC
)

var (
	JsonFormat = false
	// if set to production, this will replace some errors to warning that can be caused by misuse instead of bugs
	ExtendedLogLevel = "development"
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

func LoggingLevel(logLevel string) {
	switch logLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	LavaFormatInfo("setting log level", Attribute{Key: "loglevel", Value: logLevel})
}

func LavaFormatLog(description string, err error, attributes []Attribute, severity uint) error {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	NoColor := true

	if JsonFormat {
		zerologlog.Logger = zerologlog.Output(os.Stderr)
	} else {
		zerologlog.Logger = zerologlog.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: NoColor, TimeFormat: time.Stamp})
	}

	var logEvent *zerolog.Event
	switch severity {
	case LAVA_LOG_PANIC:
		// prefix = "Panic:"
		logEvent = zerologlog.Panic()
	case LAVA_LOG_FATAL:
		// prefix = "Fatal:"
		logEvent = zerologlog.Fatal()
	case LAVA_LOG_ERROR:
		// prefix = "Error:"
		logEvent = zerologlog.Error()
	case LAVA_LOG_WARN:
		// prefix = "Warning:"
		logEvent = zerologlog.Warn()
	case LAVA_LOG_INFO:
		logEvent = zerologlog.Info()
		// prefix = "Info:"
	case LAVA_LOG_DEBUG:
		logEvent = zerologlog.Debug()
		// prefix = "Debug:"
	}
	output := description
	attrStrings := []string{}
	if err != nil {
		logEvent = logEvent.Err(err)
		output = fmt.Sprintf("%s ErrMsg: %s", output, err.Error())
	}
	if len(attributes) > 0 {
		for idx, attr := range attributes {
			key := attr.Key
			val := attr.Value
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
			case error:
				st_val = value.Error()
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
			logEvent = logEvent.Str(key, st_val)
			attrStrings = append(attrStrings, fmt.Sprintf("%s:%s", attr.Key, st_val))
		}
		attributesStr := "{" + strings.Join(attrStrings, ",") + "}"
		output = fmt.Sprintf("%s %+v", output, attributesStr)
	}
	logEvent.Msg(description)
	// here we return the same type of the original error message, this handles nil case as well
	errRet := sdkerrors.Wrap(err, output)
	if errRet == nil { // we always want to return an error if lavaFormatError was called
		return fmt.Errorf(output)
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

// depending on the build flag, this log function will log either a warning or an error.
// the purpose of this function is to fail E2E tests and not allow unexpected behavior to reach main.
// while in production some errors may occur as consumers / providers might set up their processes in the wrong way.
// in test environment we dont expect to have these errors and if they occur we would like to fail the test.
func LavaFormatProduction(description string, err error, attributes ...Attribute) error {
	if ExtendedLogLevel == "production" {
		return LavaFormatWarning(description, err, attributes...)
	} else {
		return LavaFormatError(description, err, attributes...)
	}
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

func FormatStringerList[T fmt.Stringer](description string, listToPrint []T) string {
	st := ""
	for _, printable := range listToPrint {
		st = st + printable.String() + "\n"
	}
	st = fmt.Sprintf(description+"\n\t%s", st)
	return st
}
