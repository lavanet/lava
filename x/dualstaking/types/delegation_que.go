package types

// mark the next to be a regular delegation/unbond, as stand alone we want to take action
func (dq *DelegationQue) DelegateUnbond() {
	dq.Que = append(dq.Que, true)
}

// redelegates does unbond and than redelegate, we dont want to take action in dual staking, mark them as false
func (dq *DelegationQue) Redelegate() {
	dq.Que = append(dq.Que, false)
	dq.Que = append(dq.Que, false)
}

func (dq *DelegationQue) DeQue() bool {
	item := dq.Que[0]
	dq.Que = dq.Que[1:]
	return item
}
