package types

const (
	STALE_ENTRY_TIME int64 = 1440 // 1440 blocks (equivalent to 24 hours when block_time = 1min)
)

// References action enum
const (
	ADD_REFERENCE = 0
	SUB_REFERENCE = 1
	DO_NOTHING    = 2
)
