package utils

import (
	"errors"
	"fmt"

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

func LavaFormatLog(err error, description string, extraAttributes *map[string]any, severity uint) error {
	var prefix string
	switch severity {
	case 2:
		prefix = "Error:"
	case 1:
		prefix = "Warning:"
	case 0:
		prefix = "Info:"
	}
	output := prefix + " " + description
	if err != nil {
		output = fmt.Sprintf("%s Error: %s", output, err.Error())
	}
	if extraAttributes != nil {
		output = fmt.Sprintf("%s Extra: %v", output, extraAttributes)
	}
	fmt.Printf(output)
	return nil
}

func LavaFormatError(err error, description string, extraAttributes *map[string]any) error {
	return LavaFormatLog(err, description, extraAttributes, 2)
}

func LavaFormatWarning(err error, description string, extraAttributes *map[string]any) error {
	return LavaFormatLog(err, description, extraAttributes, 1)
}

func LavaFormatInfo(err error, description string, extraAttributes *map[string]any) error {
	return LavaFormatLog(err, description, extraAttributes, 0)
}
