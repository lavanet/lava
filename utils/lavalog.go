package utils

import (
	"errors"
	"fmt"
	"os"

	golog "log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	EventPrefix = "lava_"
)

func LogLavaEvent(ctx sdk.Context, logger log.Logger, name string, attributes map[string]string, description string) {
	attributes_str := ""
	eventAttrs := []sdk.Attribute{}
	for key, val := range attributes {
		attributes_str += fmt.Sprintf("%s: %s,", key, val)
		eventAttrs = append(eventAttrs, sdk.NewAttribute(key, val))
	}
	logger.Info(fmt.Sprintf("%s%s: %s %s", EventPrefix, name, description, attributes_str))
	ctx.EventManager().EmitEvent(sdk.NewEvent(EventPrefix+name, eventAttrs...))
}

func LavaError(ctx sdk.Context, logger log.Logger, name string, attributes map[string]string, description string) error {
	attributes_str := ""
	eventAttrs := []sdk.Attribute{}
	for key, val := range attributes {
		attributes_str += fmt.Sprintf("%s: %s,", key, val)
		eventAttrs = append(eventAttrs, sdk.NewAttribute(key, val))
	}
	err_msg := fmt.Sprintf("ERR_%s: %s %s", name, description, attributes_str)
	logger.Error(err_msg)
	// ctx.EventManager().EmitEvent(sdk.NewEvent("ERR_"+name, eventAttrs...))
	//TODO: add error types, create them here and return
	return errors.New(err_msg)
}

func LavaFormatLog(description string, err error, extraAttributes *map[string]string, severity uint) error {
	var prefix string
	switch severity {
	case 3:
		prefix = "Fatal:"
	case 2:
		prefix = "Error:"
	case 1:
		prefix = "Warning:"
	case 0:
		prefix = "Info:"
	}
	output := prefix + " " + description
	if err != nil {
		output = fmt.Sprintf("%s ErrMsg: %s", output, err.Error())
	}
	if extraAttributes != nil {
		output = fmt.Sprintf("%s -- %v", output, extraAttributes)
	}
	golog.Println(output)
	return fmt.Errorf(output)
}

func LavaFormatFatal(description string, err error, extraAttributes *map[string]string) {
	LavaFormatLog(description, err, extraAttributes, 3)
	os.Exit(1)
}

func LavaFormatError(description string, err error, extraAttributes *map[string]string) error {
	return LavaFormatLog(description, err, extraAttributes, 2)
}

func LavaFormatWarning(description string, err error, extraAttributes *map[string]string) error {
	return LavaFormatLog(description, err, extraAttributes, 1)
}

func LavaFormatInfo(description string, extraAttributes *map[string]string) error {
	return LavaFormatLog(description, nil, extraAttributes, 0)
}
