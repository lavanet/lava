package common

import "github.com/lavanet/lava/v5/utils"

// LogCodedError logs an error with a classified LavaError, auto-populating structured fields.
func LogCodedError(description string, err error, lavaError *LavaError, attributes ...utils.Attribute) error {
	if lavaError == nil {
		lavaError = LavaErrorUnknown
	}
	return utils.LavaFormatCodedError(
		description,
		err,
		lavaError.Code,
		lavaError.Name,
		lavaError.Category.String(),
		lavaError.Retryable,
		attributes...,
	)
}
