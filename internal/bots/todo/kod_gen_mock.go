//go:build !ignoreKodGen

// Code generated by MockGen. DO NOT EDIT.
// Source: internal/bots/todo/kod_gen_interface.go
//
// Generated by this command:
//
//	mockgen -source internal/bots/todo/kod_gen_interface.go -destination internal/bots/todo/kod_gen_mock.go -package todo -typed -build_constraint !ignoreKodGen
//

// Package todo is a generated GoMock package.
package todo

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockTodoBot is a mock of TodoBot interface.
type MockTodoBot struct {
	ctrl     *gomock.Controller
	recorder *MockTodoBotMockRecorder
	isgomock struct{}
}

// MockTodoBotMockRecorder is the mock recorder for MockTodoBot.
type MockTodoBotMockRecorder struct {
	mock *MockTodoBot
}

// NewMockTodoBot creates a new mock instance.
func NewMockTodoBot(ctrl *gomock.Controller) *MockTodoBot {
	mock := &MockTodoBot{ctrl: ctrl}
	mock.recorder = &MockTodoBotMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTodoBot) EXPECT() *MockTodoBotMockRecorder {
	return m.recorder
}

// Name mocks base method.
func (m *MockTodoBot) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockTodoBotMockRecorder) Name() *MockTodoBotNameCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockTodoBot)(nil).Name))
	return &MockTodoBotNameCall{Call: call}
}

// MockTodoBotNameCall wrap *gomock.Call
type MockTodoBotNameCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockTodoBotNameCall) Return(arg0 string) *MockTodoBotNameCall {
	c.Call = c.Call.Return(arg0)
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockTodoBotNameCall) Do(f func() string) *MockTodoBotNameCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockTodoBotNameCall) DoAndReturn(f func() string) *MockTodoBotNameCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}

// Start mocks base method.
func (m *MockTodoBot) Start() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Start")
}

// Start indicates an expected call of Start.
func (mr *MockTodoBotMockRecorder) Start() *MockTodoBotStartCall {
	mr.mock.ctrl.T.Helper()
	call := mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockTodoBot)(nil).Start))
	return &MockTodoBotStartCall{Call: call}
}

// MockTodoBotStartCall wrap *gomock.Call
type MockTodoBotStartCall struct {
	*gomock.Call
}

// Return rewrite *gomock.Call.Return
func (c *MockTodoBotStartCall) Return() *MockTodoBotStartCall {
	c.Call = c.Call.Return()
	return c
}

// Do rewrite *gomock.Call.Do
func (c *MockTodoBotStartCall) Do(f func()) *MockTodoBotStartCall {
	c.Call = c.Call.Do(f)
	return c
}

// DoAndReturn rewrite *gomock.Call.DoAndReturn
func (c *MockTodoBotStartCall) DoAndReturn(f func()) *MockTodoBotStartCall {
	c.Call = c.Call.DoAndReturn(f)
	return c
}
