package core

import (
	"context"
	"testing"

	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/modules/resource/core/mocks"
	"github.com/liny/sim-hub/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestUseCase() (*UseCase, *mocks.MockBlobStore, *mocks.MockSTSProvider, *gorm.DB) {
	// Initialize in-memory SQLite for testing
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	// Create required tables (Simplified for Resource module)
	// In real project, you would auto-migrate all needed models
	// For this test, RequestUploadToken doesn't need DB yet, but others do.

	d := &data.Data{DB: db}
	mockStore := new(mocks.MockBlobStore)
	mockSTS := new(mocks.MockSTSProvider)

	uc := NewUseCase(d, mockStore, mockSTS, "test-bucket", nil, "combined", "http://localhost:30030", nil)
	return uc, mockStore, mockSTS, db
}

func TestRequestUploadToken(t *testing.T) {
	uc, mockStore, _, _ := setupTestUseCase()

	t.Run("Default Presigned URL Mode", func(t *testing.T) {
		req := ApplyUploadTokenRequest{
			ResourceType: "scenario",
			Filename:     "test.zip",
			Mode:         "presigned",
		}

		// Expect PresignPut to be called
		expectedURL := "http://mock-minio/test.zip?signature=xxx"
		mockStore.On("PresignPut", mock.Anything, "test-bucket", mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).
			Return(expectedURL, nil).Once()

		ticket, err := uc.RequestUploadToken(context.Background(), req)

		assert.NoError(t, err)
		assert.Equal(t, expectedURL, ticket.PresignedURL)
		assert.Contains(t, ticket.TicketID, "resources/scenario/")
		mockStore.AssertExpectations(t)
	})

	t.Run("STS Mode", func(t *testing.T) {
		uc, _, mockSTS, _ := setupTestUseCase()
		req := ApplyUploadTokenRequest{
			ResourceType: "scenario",
			Filename:     "test.zip",
			Mode:         "sts",
		}

		expectedCreds := &storage.STSCredentials{
			AccessKey:    "AK",
			SecretKey:    "SK",
			SessionToken: "ST",
		}
		mockSTS.On("GenerateSTSToken", mock.Anything, "test-bucket", mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).
			Return(expectedCreds, nil).Once()

		ticket, err := uc.RequestUploadToken(context.Background(), req)

		assert.NoError(t, err)
		assert.Equal(t, "AK", ticket.Credentials.AccessKey)
		assert.NotNil(t, ticket.Credentials)
		mockSTS.AssertExpectations(t)
	})
}
