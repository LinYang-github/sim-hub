package core

import (
	"context"
	"testing"

	"sim-hub/internal/data"
	"sim-hub/internal/model"
	"sim-hub/internal/modules/resource/core/mocks"
	"sim-hub/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestUseCase() (*UseCase, *mocks.MockBlobStore, *mocks.MockSTSProvider, *gorm.DB) {
	// Initialize in-memory SQLite for testing
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	d := &data.Data{DB: db}
	// Migrate all needed models for integration-like testing
	db.AutoMigrate(
		&model.Resource{},
		&model.ResourceVersion{},
		&model.ResourceDependency{},
	)

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

func TestDependencyResolution(t *testing.T) {
	uc, _, _, db := setupTestUseCase()

	// 1. Prepare data: Resource A -> Resource B -> Resource C
	resC := model.Resource{Name: "Resource C", TypeKey: "model"}
	db.Create(&resC)
	verC := model.ResourceVersion{ResourceID: resC.ID, VersionNum: 1, SemVer: "v1.0.0", State: "ACTIVE"}
	db.Create(&verC)

	resB := model.Resource{Name: "Resource B", TypeKey: "model"}
	db.Create(&resB)
	verB := model.ResourceVersion{ResourceID: resB.ID, VersionNum: 1, SemVer: "v1.1.0", State: "ACTIVE"}
	db.Create(&verB)
	db.Create(&model.ResourceDependency{SourceVersionID: verB.ID, TargetResourceID: resC.ID, Constraint: "latest"})

	resA := model.Resource{Name: "Resource A", TypeKey: "scenario"}
	db.Create(&resA)
	verA := model.ResourceVersion{ResourceID: resA.ID, VersionNum: 1, SemVer: "v2.0.0", State: "ACTIVE"}
	db.Create(&verA)
	db.Create(&model.ResourceDependency{SourceVersionID: verA.ID, TargetResourceID: resB.ID, Constraint: "latest"})

	t.Run("Resolve Flat Dependencies", func(t *testing.T) {
		deps, err := uc.GetResourceDependencies(context.Background(), verA.ID)
		assert.NoError(t, err)
		assert.Len(t, deps, 1)
		assert.Equal(t, resB.ID, deps[0].TargetResourceID)
	})

	t.Run("Resolve Recursive Dependency Tree", func(t *testing.T) {
		tree, err := uc.GetDependencyTree(context.Background(), verA.ID)
		assert.NoError(t, err)
		assert.Len(t, tree, 1)

		nodeB := tree[0]
		assert.Equal(t, "Resource B", nodeB["resource_name"])
		assert.Equal(t, "v1.1.0", nodeB["semver"])

		children := nodeB["dependencies"].([]map[string]any)
		assert.Len(t, children, 1)
		assert.Equal(t, "Resource C", children[0]["resource_name"])
	})

	t.Run("Cycle Detection Safety", func(t *testing.T) {
		// Create a cycle: C -> A
		db.Create(&model.ResourceDependency{SourceVersionID: verC.ID, TargetResourceID: resA.ID, Constraint: "latest"})

		tree, err := uc.GetDependencyTree(context.Background(), verA.ID)
		assert.NoError(t, err)
		assert.NotNil(t, tree)
		// Should not stack overflow
	})
}
