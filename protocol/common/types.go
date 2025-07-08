package common

// QuorumParams holds the quorum configuration parameters
type QuorumParams struct {
	Rate float64
	Max  int
	Min  int
}

func (qp QuorumParams) Enabled() bool {
	return qp.Rate == 1 && qp.Max == 1 && qp.Min == 1
}
