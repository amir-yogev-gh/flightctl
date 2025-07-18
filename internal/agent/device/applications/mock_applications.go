// Code generated by MockGen. DO NOT EDIT.
// Source: applications.go
//
// Generated by this command:
//
//	mockgen -source=applications.go -destination=mock_applications.go -package=applications
//

// Package applications is a generated GoMock package.
package applications

import (
	context "context"
	reflect "reflect"

	v1alpha1 "github.com/flightctl/flightctl/api/v1alpha1"
	provider "github.com/flightctl/flightctl/internal/agent/device/applications/provider"
	dependency "github.com/flightctl/flightctl/internal/agent/device/dependency"
	status "github.com/flightctl/flightctl/internal/agent/device/status"
	gomock "go.uber.org/mock/gomock"
)

// MockMonitor is a mock of Monitor interface.
type MockMonitor struct {
	ctrl     *gomock.Controller
	recorder *MockMonitorMockRecorder
}

// MockMonitorMockRecorder is the mock recorder for MockMonitor.
type MockMonitorMockRecorder struct {
	mock *MockMonitor
}

// NewMockMonitor creates a new mock instance.
func NewMockMonitor(ctrl *gomock.Controller) *MockMonitor {
	mock := &MockMonitor{ctrl: ctrl}
	mock.recorder = &MockMonitorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMonitor) EXPECT() *MockMonitorMockRecorder {
	return m.recorder
}

// Run mocks base method.
func (m *MockMonitor) Run(ctx context.Context) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Run", ctx)
}

// Run indicates an expected call of Run.
func (mr *MockMonitorMockRecorder) Run(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockMonitor)(nil).Run), ctx)
}

// Status mocks base method.
func (m *MockMonitor) Status() []v1alpha1.DeviceApplicationStatus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Status")
	ret0, _ := ret[0].([]v1alpha1.DeviceApplicationStatus)
	return ret0
}

// Status indicates an expected call of Status.
func (mr *MockMonitorMockRecorder) Status() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Status", reflect.TypeOf((*MockMonitor)(nil).Status))
}

// MockManager is a mock of Manager interface.
type MockManager struct {
	ctrl     *gomock.Controller
	recorder *MockManagerMockRecorder
}

// MockManagerMockRecorder is the mock recorder for MockManager.
type MockManagerMockRecorder struct {
	mock *MockManager
}

// NewMockManager creates a new mock instance.
func NewMockManager(ctrl *gomock.Controller) *MockManager {
	mock := &MockManager{ctrl: ctrl}
	mock.recorder = &MockManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockManager) EXPECT() *MockManagerMockRecorder {
	return m.recorder
}

// AfterUpdate mocks base method.
func (m *MockManager) AfterUpdate(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AfterUpdate", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// AfterUpdate indicates an expected call of AfterUpdate.
func (mr *MockManagerMockRecorder) AfterUpdate(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AfterUpdate", reflect.TypeOf((*MockManager)(nil).AfterUpdate), ctx)
}

// BeforeUpdate mocks base method.
func (m *MockManager) BeforeUpdate(ctx context.Context, desired *v1alpha1.DeviceSpec) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BeforeUpdate", ctx, desired)
	ret0, _ := ret[0].(error)
	return ret0
}

// BeforeUpdate indicates an expected call of BeforeUpdate.
func (mr *MockManagerMockRecorder) BeforeUpdate(ctx, desired any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BeforeUpdate", reflect.TypeOf((*MockManager)(nil).BeforeUpdate), ctx, desired)
}

// CollectOCITargets mocks base method.
func (m *MockManager) CollectOCITargets(ctx context.Context, current, desired *v1alpha1.DeviceSpec) ([]dependency.OCIPullTarget, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CollectOCITargets", ctx, current, desired)
	ret0, _ := ret[0].([]dependency.OCIPullTarget)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CollectOCITargets indicates an expected call of CollectOCITargets.
func (mr *MockManagerMockRecorder) CollectOCITargets(ctx, current, desired any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CollectOCITargets", reflect.TypeOf((*MockManager)(nil).CollectOCITargets), ctx, current, desired)
}

// Ensure mocks base method.
func (m *MockManager) Ensure(ctx context.Context, provider provider.Provider) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Ensure", ctx, provider)
	ret0, _ := ret[0].(error)
	return ret0
}

// Ensure indicates an expected call of Ensure.
func (mr *MockManagerMockRecorder) Ensure(ctx, provider any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Ensure", reflect.TypeOf((*MockManager)(nil).Ensure), ctx, provider)
}

// Remove mocks base method.
func (m *MockManager) Remove(ctx context.Context, provider provider.Provider) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Remove", ctx, provider)
	ret0, _ := ret[0].(error)
	return ret0
}

// Remove indicates an expected call of Remove.
func (mr *MockManagerMockRecorder) Remove(ctx, provider any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Remove", reflect.TypeOf((*MockManager)(nil).Remove), ctx, provider)
}

// Status mocks base method.
func (m *MockManager) Status(arg0 context.Context, arg1 *v1alpha1.DeviceStatus, arg2 ...status.CollectorOpt) error {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Status", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Status indicates an expected call of Status.
func (mr *MockManagerMockRecorder) Status(arg0, arg1 any, arg2 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Status", reflect.TypeOf((*MockManager)(nil).Status), varargs...)
}

// Stop mocks base method.
func (m *MockManager) Stop(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockManagerMockRecorder) Stop(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockManager)(nil).Stop), ctx)
}

// Update mocks base method.
func (m *MockManager) Update(ctx context.Context, provider provider.Provider) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, provider)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockManagerMockRecorder) Update(ctx, provider any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockManager)(nil).Update), ctx, provider)
}

// MockApplication is a mock of Application interface.
type MockApplication struct {
	ctrl     *gomock.Controller
	recorder *MockApplicationMockRecorder
}

// MockApplicationMockRecorder is the mock recorder for MockApplication.
type MockApplicationMockRecorder struct {
	mock *MockApplication
}

// NewMockApplication creates a new mock instance.
func NewMockApplication(ctrl *gomock.Controller) *MockApplication {
	mock := &MockApplication{ctrl: ctrl}
	mock.recorder = &MockApplicationMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockApplication) EXPECT() *MockApplicationMockRecorder {
	return m.recorder
}

// AddWorkload mocks base method.
func (m *MockApplication) AddWorkload(Workload *Workload) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddWorkload", Workload)
}

// AddWorkload indicates an expected call of AddWorkload.
func (mr *MockApplicationMockRecorder) AddWorkload(Workload any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddWorkload", reflect.TypeOf((*MockApplication)(nil).AddWorkload), Workload)
}

// AppType mocks base method.
func (m *MockApplication) AppType() v1alpha1.AppType {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AppType")
	ret0, _ := ret[0].(v1alpha1.AppType)
	return ret0
}

// AppType indicates an expected call of AppType.
func (mr *MockApplicationMockRecorder) AppType() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AppType", reflect.TypeOf((*MockApplication)(nil).AppType))
}

// ID mocks base method.
func (m *MockApplication) ID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockApplicationMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockApplication)(nil).ID))
}

// IsEmbedded mocks base method.
func (m *MockApplication) IsEmbedded() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsEmbedded")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsEmbedded indicates an expected call of IsEmbedded.
func (mr *MockApplicationMockRecorder) IsEmbedded() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsEmbedded", reflect.TypeOf((*MockApplication)(nil).IsEmbedded))
}

// Name mocks base method.
func (m *MockApplication) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockApplicationMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockApplication)(nil).Name))
}

// Path mocks base method.
func (m *MockApplication) Path() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Path")
	ret0, _ := ret[0].(string)
	return ret0
}

// Path indicates an expected call of Path.
func (mr *MockApplicationMockRecorder) Path() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Path", reflect.TypeOf((*MockApplication)(nil).Path))
}

// RemoveWorkload mocks base method.
func (m *MockApplication) RemoveWorkload(name string) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveWorkload", name)
	ret0, _ := ret[0].(bool)
	return ret0
}

// RemoveWorkload indicates an expected call of RemoveWorkload.
func (mr *MockApplicationMockRecorder) RemoveWorkload(name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveWorkload", reflect.TypeOf((*MockApplication)(nil).RemoveWorkload), name)
}

// Status mocks base method.
func (m *MockApplication) Status() (*v1alpha1.DeviceApplicationStatus, v1alpha1.DeviceApplicationsSummaryStatus, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Status")
	ret0, _ := ret[0].(*v1alpha1.DeviceApplicationStatus)
	ret1, _ := ret[1].(v1alpha1.DeviceApplicationsSummaryStatus)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Status indicates an expected call of Status.
func (mr *MockApplicationMockRecorder) Status() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Status", reflect.TypeOf((*MockApplication)(nil).Status))
}

// Volume mocks base method.
func (m *MockApplication) Volume() provider.VolumeManager {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Volume")
	ret0, _ := ret[0].(provider.VolumeManager)
	return ret0
}

// Volume indicates an expected call of Volume.
func (mr *MockApplicationMockRecorder) Volume() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Volume", reflect.TypeOf((*MockApplication)(nil).Volume))
}

// Workload mocks base method.
func (m *MockApplication) Workload(name string) (*Workload, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Workload", name)
	ret0, _ := ret[0].(*Workload)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// Workload indicates an expected call of Workload.
func (mr *MockApplicationMockRecorder) Workload(name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Workload", reflect.TypeOf((*MockApplication)(nil).Workload), name)
}
