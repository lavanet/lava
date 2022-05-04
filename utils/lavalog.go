package utils

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"
)

func LogLavaEvent(ctx sdk.Context, logger log.Logger, name string, attributes map[string]string, description string) {
	attributes_str := ""
	eventAttrs := []sdk.Attribute{}
	for key, val := range attributes {
		attributes_str += fmt.Sprintf("%s: %s,", key, val)
		eventAttrs = append(eventAttrs, sdk.NewAttribute(key, val))
	}
	logger.Info(fmt.Sprintf("lava_%s: %s %s", name, description, attributes_str))
	ctx.EventManager().EmitEvent(sdk.NewEvent("lava_"+name, eventAttrs...))
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
