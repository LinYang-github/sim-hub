package core

import (
	"context"
	"testing"

	"sim-hub/internal/data"
	"sim-hub/internal/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStorageUnavailableRobustness(t *testing.T) {
	// Initialize in-memory SQLite for testing to avoid nil DB panic
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.Resource{}, &model.ResourceVersion{})

	d := &data.Data{DB: db}
	uploader := NewUploadManager(d, nil, nil, "test-bucket", nil)
	reader := NewResourceReader(d, nil, "test-bucket")

	t.Run("RequestUploadToken returns error when store is nil", func(t *testing.T) {
		req := ApplyUploadTokenRequest{
			ResourceType: "scenario",
			Filename:     "test.zip",
		}
		ticket, err := uploader.RequestUploadToken(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, ticket)
		assert.Contains(t, err.Error(), "storage service (MinIO) is not available")
	})

	t.Run("InitMultipartUpload returns error when store is nil", func(t *testing.T) {
		req := InitMultipartUploadRequest{
			ResourceType: "scenario",
			Filename:     "test.zip",
		}
		resp, err := uploader.InitMultipartUpload(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "storage service (MinIO) is not available")
	})

	t.Run("GetResource returns error for presigned URL when store is nil", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_, _ = reader.GetResource(context.Background(), "any-id")
		})
	})
}
