package rpcsmartrouter

import (
	"reflect"

	gomock "go.uber.org/mock/gomock"
)

// MinimalStateTrackerInf is a minimal interface for smart router state tracking
type MinimalStateTrackerInf interface {
	GetLatestVirtualEpoch() uint64
}

// MockMinimalStateTrackerInf is a mock of MinimalStateTrackerInf interface.
type MockMinimalStateTrackerInf struct {
	ctrl     *gomock.Controller
	recorder *MockMinimalStateTrackerInfMockRecorder
}

// MockMinimalStateTrackerInfMockRecorder is the mock recorder for MockMinimalStateTrackerInf.
type MockMinimalStateTrackerInfMockRecorder struct {
	mock *MockMinimalStateTrackerInf
}

// NewMockMinimalStateTrackerInf creates a new mock instance.
func NewMockMinimalStateTrackerInf(ctrl *gomock.Controller) *MockMinimalStateTrackerInf {
	mock := &MockMinimalStateTrackerInf{ctrl: ctrl}
	mock.recorder = &MockMinimalStateTrackerInfMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMinimalStateTrackerInf) EXPECT() *MockMinimalStateTrackerInfMockRecorder {
	return m.recorder
}

// GetLatestVirtualEpoch mocks base method.
func (m *MockMinimalStateTrackerInf) GetLatestVirtualEpoch() uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLatestVirtualEpoch")
	ret0, _ := ret[0].(uint64)
	return ret0
}

// GetLatestVirtualEpoch indicates an expected call of GetLatestVirtualEpoch.
func (mr *MockMinimalStateTrackerInfMockRecorder) GetLatestVirtualEpoch() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLatestVirtualEpoch", reflect.TypeOf((*MockMinimalStateTrackerInf)(nil).GetLatestVirtualEpoch))
}
