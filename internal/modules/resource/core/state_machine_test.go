package core

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/model"
	"github.com/liny/sim-hub/internal/modules/resource/core/mocks"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestVersionStateMachineDepth(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.Resource{}, &model.ResourceVersion{})
	d := &data.Data{DB: db}

	storageMock := new(mocks.MockBlobStore)
	dispatcher := &MockDispatcher{}
	// Mock EventEmitter might need a real-ish nats client or we mock it.
	// For simplicity, we use nil nats in NewEventEmitter which won't publish but will log.
	emitter := NewEventEmitter(nil)

	writer := NewResourceWriter(d, storageMock, "test-bucket", dispatcher, emitter, nil)

	t.Run("Status Transition: PENDING -> ACTIVE with Metadata Merge", func(t *testing.T) {
		resID := uuid.New().String()
		verID := uuid.New().String()

		db.Create(&model.Resource{ID: resID, Name: "state-test"})
		db.Create(&model.ResourceVersion{
			ID:         verID,
			ResourceID: resID,
			State:      "PENDING",
			MetaData:   map[string]any{"initial": "val"},
		})

		req := ProcessResultRequest{
			State:    "ACTIVE",
			MetaData: map[string]any{"extracted": "data"},
		}

		err := writer.ReportProcessResult(context.Background(), verID, req)
		assert.NoError(t, err)

		var ver model.ResourceVersion
		db.First(&ver, "id = ?", verID)
		assert.Equal(t, "ACTIVE", ver.State)
		assert.Equal(t, "val", ver.MetaData["initial"])
		assert.Equal(t, "data", ver.MetaData["extracted"])

		// Should dispatch Refresh job
		assert.GreaterOrEqual(t, len(dispatcher.DispatchedJobs), 1)
		lastJob := dispatcher.DispatchedJobs[len(dispatcher.DispatchedJobs)-1]
		assert.Equal(t, ActionRefresh, lastJob.Action)
		assert.Equal(t, verID, lastJob.VersionID)
	})

	t.Run("Status Transition: PENDING -> ERROR", func(t *testing.T) {
		verID := uuid.New().String()
		db.Create(&model.ResourceVersion{
			ID:    verID,
			State: "PENDING",
		})

		req := ProcessResultRequest{
			State:   "ERROR",
			Message: "Process failed",
		}

		err := writer.ReportProcessResult(context.Background(), verID, req)
		assert.NoError(t, err)

		var ver model.ResourceVersion
		db.First(&ver, "id = ?", verID)
		assert.Equal(t, "ERROR", ver.State)
		// Should NOT dispatch Refresh job on error
	})
}
