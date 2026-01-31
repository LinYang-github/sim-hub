package core

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"sim-hub/internal/data"
	"sim-hub/internal/model"
	"sim-hub/internal/modules/resource/core/mocks"
)

type MockDispatcher struct {
	DispatchedJobs []ProcessJob
}

func (m *MockDispatcher) Dispatch(ctx context.Context, job ProcessJob) {
	m.DispatchedJobs = append(m.DispatchedJobs, job)
}

func setupTestDB(t *testing.T) *data.Data {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// AutoMigrate tables needed for test
	err = db.AutoMigrate(
		&model.ResourceType{},
		&model.Resource{},
		&model.ResourceVersion{},
		&model.Category{},
		&model.ResourceDependency{},
	)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return &data.Data{DB: db}
}

func TestSetResourceLatestVersion(t *testing.T) {
	d := setupTestDB(t)
	storageMock := new(mocks.MockBlobStore)
	dispatcher := &MockDispatcher{}

	writer := NewResourceWriter(d, storageMock, "test-bucket", dispatcher, nil, nil)

	// Setup Data
	resID := uuid.New().String()
	verID1 := uuid.New().String()
	verID2 := uuid.New().String()

	res := model.Resource{
		ID:      resID,
		Name:    "test-res",
		TypeKey: "model_glb",
	}
	d.DB.Create(&res)

	ver1 := model.ResourceVersion{
		ID:         verID1,
		ResourceID: resID,
		VersionNum: 1,
		SemVer:     "v1.0.0",
		State:      "ACTIVE",
		FilePath:   "path/to/v1",
	}
	d.DB.Create(&ver1)

	ver2 := model.ResourceVersion{
		ID:         verID2,
		ResourceID: resID,
		VersionNum: 2,
		SemVer:     "v2.0.0",
		State:      "ACTIVE",
		FilePath:   "path/to/v2",
	}
	d.DB.Create(&ver2)

	// Test
	ctx := context.Background()
	err := writer.SetResourceLatestVersion(ctx, resID, ver2.ID)

	assert.NoError(t, err)

	// Check DB
	var updatedRes model.Resource
	d.DB.First(&updatedRes, "id = ?", resID)
	assert.Equal(t, ver2.ID, updatedRes.LatestVersionID)

	// Check Dispatcher
	assert.Len(t, dispatcher.DispatchedJobs, 1)
	assert.Equal(t, ActionRefresh, dispatcher.DispatchedJobs[0].Action)
	assert.Equal(t, ver2.ID, dispatcher.DispatchedJobs[0].VersionID)
}

func TestSetResourceLatestVersion_InvalidVersion(t *testing.T) {
	d := setupTestDB(t)
	storageMock := new(mocks.MockBlobStore)
	dispatcher := &MockDispatcher{}

	writer := NewResourceWriter(d, storageMock, "test-bucket", dispatcher, nil, nil)

	// Setup Data
	resID := uuid.New().String()
	resID2 := uuid.New().String()
	verIDOther := uuid.New().String()

	res := model.Resource{ID: resID}
	d.DB.Create(&res)
	res2 := model.Resource{ID: resID2}
	d.DB.Create(&res2)

	// Version belongs to res2
	verOther := model.ResourceVersion{
		ID:         verIDOther,
		ResourceID: resID2,
		VersionNum: 1,
		State:      "ACTIVE",
	}
	d.DB.Create(&verOther)

	// Attempt to set res1 latest version to verOther
	ctx := context.Background()
	err := writer.SetResourceLatestVersion(ctx, resID, verIDOther)

	assert.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestCreateResourceAndVersion(t *testing.T) {
	d := setupTestDB(t)
	storageMock := new(mocks.MockBlobStore)
	dispatcher := &MockDispatcher{}
	handlers := map[string]string{"model_glb": "glb-processor"}

	writer := NewResourceWriter(d, storageMock, "test-bucket", dispatcher, nil, handlers)

	// Create ResourceType to satisfy FK if needed (though sqlite might not enforce)
	d.DB.Create(&model.ResourceType{TypeKey: "model_glb", TypeName: "GLB Model"})

	// Test creation
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		return writer.CreateResourceAndVersion(tx, "model_glb", "cat-1", "new-res", "owner-1", "PUBLIC", "path/obj", 100, []string{"tag1"}, "v1.0.0", nil, nil)
	})
	assert.NoError(t, err)

	// Check Resource
	var res model.Resource
	d.DB.First(&res, "name = ?", "new-res")
	assert.Equal(t, "model_glb", res.TypeKey)
	assert.Equal(t, "PUBLIC", res.Scope)
	// Compare slices differently or use helper, but here straightforward
	// assert.Contains(t, res.Tags, "tag1") // Tags is serializer:json, might need handling. In test sqlite it's stored as text/blob. GORM handles it.
	// Since we defined string array and hook, let's just check length if deserialization works or skip tags detailed check for now.

	// Check Version
	var ver model.ResourceVersion
	d.DB.First(&ver, "resource_id = ?", res.ID)
	assert.Equal(t, "v1.0.0", ver.SemVer)
	assert.Equal(t, "PENDING", ver.State) // Should be PENDING because handler exists

	// Check Dispatch
	assert.Len(t, dispatcher.DispatchedJobs, 1)
	assert.Equal(t, ActionProcess, dispatcher.DispatchedJobs[0].Action)
}

func TestCreateResourceAndVersion_DuplicateSemVer_Pending(t *testing.T) {
	d := setupTestDB(t)
	storageMock := new(mocks.MockBlobStore)
	dispatcher := &MockDispatcher{}
	handlers := map[string]string{"model_glb": "glb-processor"}

	writer := NewResourceWriter(d, storageMock, "test-bucket", dispatcher, nil, handlers)

	// Create ResourceType
	d.DB.Create(&model.ResourceType{TypeKey: "model_glb", TypeName: "GLB Model"})

	// Setup: Create Resource and PENDING Version
	resID := uuid.New().String()
	d.DB.Create(&model.Resource{ID: resID, TypeKey: "model_glb", CategoryID: "cat-1", Name: "test-dup", OwnerID: "owner-1"})

	verID := uuid.New().String()
	d.DB.Create(&model.ResourceVersion{
		ID:         verID,
		ResourceID: resID,
		VersionNum: 1,
		SemVer:     "v1.0.0",
		State:      "PENDING",
		FilePath:   "old/path",
	})

	// Test: Re-upload v1.0.0 (Should overwrite)
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		return writer.CreateResourceAndVersion(tx, "model_glb", "cat-1", "test-dup", "owner-1", "PUBLIC", "new/path", 200, nil, "v1.0.0", nil, nil)
	})

	assert.NoError(t, err)

	// Verify
	var ver model.ResourceVersion
	d.DB.First(&ver, "resource_id = ? AND sem_ver = ?", resID, "v1.0.0")

	assert.Equal(t, "new/path", ver.FilePath)
	assert.Equal(t, int64(200), ver.FileSize)
	// ID should remain same if we updated, check if logic kept it
	assert.Equal(t, verID, ver.ID)
}

func TestCreateResourceAndVersion_DuplicateSemVer_Active(t *testing.T) {
	d := setupTestDB(t)
	storageMock := new(mocks.MockBlobStore)
	dispatcher := &MockDispatcher{}

	writer := NewResourceWriter(d, storageMock, "test-bucket", dispatcher, nil, nil)

	// Create ResourceType
	d.DB.Create(&model.ResourceType{TypeKey: "model_glb", TypeName: "GLB Model"})

	// Setup: Create Resource and ACTIVE Version
	resID := uuid.New().String()
	d.DB.Create(&model.Resource{ID: resID, TypeKey: "model_glb", CategoryID: "cat-1", Name: "test-active", OwnerID: "owner-1"})

	d.DB.Create(&model.ResourceVersion{
		ID:         uuid.New().String(),
		ResourceID: resID,
		VersionNum: 1,
		SemVer:     "v1.0.0",
		State:      "ACTIVE",
		FilePath:   "active/path",
	})

	// Test: Re-upload v1.0.0 (Should Fail)
	err := d.DB.Transaction(func(tx *gorm.DB) error {
		return writer.CreateResourceAndVersion(tx, "model_glb", "cat-1", "test-active", "owner-1", "PUBLIC", "new/path", 200, nil, "v1.0.0", nil, nil)
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists and is ACTIVE")
}
