package score

import "errors"

// sentinelError is a sentinel error type that supports the .Is(target) method
// used to identify specific error kinds in the error chain.
type sentinelError struct {
	msg string
}

func (e *sentinelError) Error() string { return e.msg }

// Is reports whether target equals this sentinel error or any error in target's chain.
func (e *sentinelError) Is(target error) bool {
	for target != nil {
		if target == error(e) {
			return true
		}
		target = errors.Unwrap(target)
	}
	return false
}

var TimeConflictingScoresError = &sentinelError{msg: "ScoreStore has a more recent sample than the one provided"}
