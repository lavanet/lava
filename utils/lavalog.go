package utils

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	zerolog "github.com/rs/zerolog"
	zerologlog "github.com/rs/zerolog/log"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	EventPrefix = "lava_"
)

var JsonFormat = false

type Attribute struct {
	Key   string
	Value interface{}
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
	case 4:
		// prefix = "Fatal:"
		logEvent = zerologlog.Fatal()

	case 3:
		// prefix = "Error:"
		logEvent = zerologlog.Error()
	case 2:
		// prefix = "Warning:"
		logEvent = zerologlog.Warn()
	case 1:
		logEvent = zerologlog.Info()
		// prefix = "Info:"
	case 0:
		logEvent = zerologlog.Debug()
		// prefix = "Debug:"
	}
	output := description
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
			case fmt.Stringer:
				st_val = value.String()
			// needs to come after stringer so byte inheriting objects will use their string method if implemented (like AccAddress)
			case []byte:
				st_val = string(value)
			case nil:
				st_val = ""
			default:
				st_val = fmt.Sprintf("%v", value)
			}
			logEvent = logEvent.Str(key, st_val)
		}
		output = fmt.Sprintf("%s -- %+v", output, attributes)
	}
	logEvent.Msg(description)
	// here we return the same type of the original error message, this handles nil case as well
	errRet := sdkerrors.Wrap(err, output)
	if errRet == nil { // we always want to return an error if lavaFormatError was called
		return fmt.Errorf(output)
	}
	return errRet
}

func LavaFormatFatal(description string, err error, attributes ...Attribute) {
	attributes = append(attributes, Attribute{Key: "StackTrace", Value: debug.Stack()})
	LavaFormatLog(description, err, attributes, 4)
	os.Exit(1)
}

func LavaFormatError(description string, err error, attributes ...Attribute) error {
	return LavaFormatLog(description, err, attributes, 3)
}

func LavaFormatWarning(description string, err error, attributes ...Attribute) error {
	return LavaFormatLog(description, err, attributes, 2)
}

func LavaFormatInfo(description string, attributes ...Attribute) error {
	return LavaFormatLog(description, nil, attributes, 1)
}

func LavaFormatDebug(description string, attributes ...Attribute) error {
	return LavaFormatLog(description, nil, attributes, 0)
}

func FormatStringerList[T fmt.Stringer](description string, listToPrint []T) string {
	st := ""
	for _, printable := range listToPrint {
		st = st + printable.String() + "\n"
	}
	st = fmt.Sprintf(description+"\n\t%s", st)
	return st
}
