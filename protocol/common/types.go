package common

// CrossValidationParams holds the cross-validation configuration parameters
type CrossValidationParams struct {
	Rate float64
	Max  int
	Min  int
}

var DefaultCrossValidationParams = CrossValidationParams{
	Rate: 1,
	Max:  1,
	Min:  1,
}

func (cvp CrossValidationParams) Enabled() bool {
	return !(cvp.Rate == DefaultCrossValidationParams.Rate && cvp.Max == DefaultCrossValidationParams.Max && cvp.Min == DefaultCrossValidationParams.Min)
}
