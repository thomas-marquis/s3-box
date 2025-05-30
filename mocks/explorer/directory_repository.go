// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/thomas-marquis/s3-box/internal/explorer (interfaces: S3DirectoryRepository)
//
// Generated by this command:
//
//	mockgen -package mocks_explorer -destination mocks/explorer/directory_repository.go github.com/thomas-marquis/s3-box/internal/explorer S3DirectoryRepository
//

// Package mocks_explorer is a generated GoMock package.
package mocks_explorer

import (
	context "context"
	reflect "reflect"

	explorer "github.com/thomas-marquis/s3-box/internal/explorer"
	gomock "go.uber.org/mock/gomock"
)

// MockS3DirectoryRepository is a mock of S3DirectoryRepository interface.
type MockS3DirectoryRepository struct {
	ctrl     *gomock.Controller
	recorder *MockS3DirectoryRepositoryMockRecorder
	isgomock struct{}
}

// MockS3DirectoryRepositoryMockRecorder is the mock recorder for MockS3DirectoryRepository.
type MockS3DirectoryRepositoryMockRecorder struct {
	mock *MockS3DirectoryRepository
}

// NewMockS3DirectoryRepository creates a new mock instance.
func NewMockS3DirectoryRepository(ctrl *gomock.Controller) *MockS3DirectoryRepository {
	mock := &MockS3DirectoryRepository{ctrl: ctrl}
	mock.recorder = &MockS3DirectoryRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockS3DirectoryRepository) EXPECT() *MockS3DirectoryRepositoryMockRecorder {
	return m.recorder
}

// GetByID mocks base method.
func (m *MockS3DirectoryRepository) GetByID(ctx context.Context, id explorer.S3DirectoryID) (*explorer.S3Directory, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, id)
	ret0, _ := ret[0].(*explorer.S3Directory)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByID indicates an expected call of GetByID.
func (mr *MockS3DirectoryRepositoryMockRecorder) GetByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockS3DirectoryRepository)(nil).GetByID), ctx, id)
}

// Save mocks base method.
func (m *MockS3DirectoryRepository) Save(ctx context.Context, d *explorer.S3Directory) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Save", ctx, d)
	ret0, _ := ret[0].(error)
	return ret0
}

// Save indicates an expected call of Save.
func (mr *MockS3DirectoryRepositoryMockRecorder) Save(ctx, d any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Save", reflect.TypeOf((*MockS3DirectoryRepository)(nil).Save), ctx, d)
}
