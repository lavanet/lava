package common

// QuorumParams holds the quorum configuration parameters
type QuorumParams struct {
	Rate float64
	Max  int
	Min  int
}

var DefaultQuorumParams = QuorumParams{
	Rate: 1,
	Max:  1,
	Min:  1,
}

func (qp QuorumParams) Enabled() bool {
	return !(qp.Rate == DefaultQuorumParams.Rate && qp.Max == DefaultQuorumParams.Max && qp.Min == DefaultQuorumParams.Min)
}
