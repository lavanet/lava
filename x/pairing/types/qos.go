package types

// NewQualityOfServiceReport creates a QualityOfServiceReport with default zero values.
func NewQualityOfServiceReport() *QualityOfServiceReport {
	return &QualityOfServiceReport{
		Latency:      0,
		Availability: 0,
		Sync:         0,
	}
}
