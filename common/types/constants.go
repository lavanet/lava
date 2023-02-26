package types

const (
	STALE_ENTRY_TIME int64 = 1440 // 1440 blocks (equivalent to 24 hours when block_time = 1min)
)

// References action enum
type ReferenceAction int32

const (
	ADD_REFERENCE ReferenceAction = 0
	SUB_REFERENCE ReferenceAction = 1
	DO_NOTHING    ReferenceAction = 2
)
