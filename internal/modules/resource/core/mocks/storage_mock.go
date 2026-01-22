package mocks

import (
	"context"
	"io"
	"time"

	"github.com/liny/sim-hub/pkg/storage"
	"github.com/stretchr/testify/mock"
)

type MockBlobStore struct {
	mock.Mock
}

func (m *MockBlobStore) Stat(ctx context.Context, bucket, key string) (*storage.ObjectInfo, error) {
	args := m.Called(ctx, bucket, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.ObjectInfo), args.Error(1)
}

func (m *MockBlobStore) Delete(ctx context.Context, bucket, key string) error {
	args := m.Called(ctx, bucket, key)
	return args.Error(0)
}

func (m *MockBlobStore) ListObjects(ctx context.Context, bucket, prefix string, recursive bool) <-chan storage.ObjectInfo {
	args := m.Called(ctx, bucket, prefix, recursive)
	return args.Get(0).(<-chan storage.ObjectInfo)
}

func (m *MockBlobStore) Put(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error {
	args := m.Called(ctx, bucket, key, reader, size, contentType)
	return args.Error(0)
}

func (m *MockBlobStore) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, bucket, key)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockBlobStore) DownloadFile(ctx context.Context, bucket, key, localPath string) error {
	args := m.Called(ctx, bucket, key, localPath)
	return args.Error(0)
}

func (m *MockBlobStore) PresignPut(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, bucket, key, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockBlobStore) PresignGet(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, bucket, key, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockBlobStore) InitMultipart(ctx context.Context, bucket, key string) (string, error) {
	args := m.Called(ctx, bucket, key)
	return args.String(0), args.Error(1)
}

func (m *MockBlobStore) PresignPart(ctx context.Context, bucket, key, uploadID string, partNumber int, expiry time.Duration) (string, error) {
	args := m.Called(ctx, bucket, key, uploadID, partNumber, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockBlobStore) CompleteMultipart(ctx context.Context, bucket, key, uploadID string, parts []storage.Part) error {
	args := m.Called(ctx, bucket, key, uploadID, parts)
	return args.Error(0)
}

func (m *MockBlobStore) AbortMultipart(ctx context.Context, bucket, key, uploadID string) error {
	args := m.Called(ctx, bucket, key, uploadID)
	return args.Error(0)
}

type MockSTSProvider struct {
	mock.Mock
}

func (m *MockSTSProvider) GenerateSTSToken(ctx context.Context, bucket, prefix string, duration time.Duration) (*storage.STSCredentials, error) {
	args := m.Called(ctx, bucket, prefix, duration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.STSCredentials), args.Error(1)
}
