package types

func (c *ResponseConflict) IsDataNil() bool {
	if c == nil {
		return true
	}
	if c.ConflictRelayData0 == nil || c.ConflictRelayData1 == nil {
		return true
	}
	if c.ConflictRelayData0.Request == nil || c.ConflictRelayData1.Request == nil {
		return true
	}
	if c.ConflictRelayData0.Request.RelayData == nil || c.ConflictRelayData1.Request.RelayData == nil {
		return true
	}
	if c.ConflictRelayData0.Request.RelaySession == nil || c.ConflictRelayData1.Request.RelaySession == nil {
		return true
	}
	if c.ConflictRelayData0.Reply == nil || c.ConflictRelayData1.Reply == nil {
		return true
	}

	return false
}
