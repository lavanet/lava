package common

// CrossValidationParams holds the cross-validation configuration parameters
// Note: Whether cross-validation is enabled is determined by the Selection type (CrossValidation),
// not by these parameters. These parameters only store the values when cross-validation is active.
type CrossValidationParams struct {
	MaxParticipants    int // Maximum number of providers to query
	AgreementThreshold int // Number of matching responses needed for consensus
}

// DefaultCrossValidationParams are used when cross-validation is not enabled (Selection != CrossValidation)
var DefaultCrossValidationParams = CrossValidationParams{
	MaxParticipants:    1,
	AgreementThreshold: 1,
}
