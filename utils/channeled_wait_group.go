package utils

import "sync"

type ChanneledWaitGroup struct {
	sync.WaitGroup
	doneChan chan struct{}
}

func NewChanneledWaitGroup() *ChanneledWaitGroup {
	return &ChanneledWaitGroup{
		doneChan: make(chan struct{}, 1),
	}
}

func (wg *ChanneledWaitGroup) Wait() <-chan struct{} {
	go func() {
		wg.WaitGroup.Wait()
		wg.doneChan <- struct{}{}
	}()
	return wg.doneChan
}
