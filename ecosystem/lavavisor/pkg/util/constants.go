package lvutil

type MismatchType int

const (
	NoMismatch MismatchType = iota
	MinVersionMismatch
	TargetVersionMismatch
)
