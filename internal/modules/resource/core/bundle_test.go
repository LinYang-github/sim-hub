package core

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"sim-hub/internal/data"
	"sim-hub/internal/model"
	"sim-hub/internal/modules/resource/core/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBundleLogicDepth(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.Resource{}, &model.ResourceVersion{}, &model.ResourceDependency{})
	d := &data.Data{DB: db}

	storageMock := new(mocks.MockBlobStore)
	reader := NewResourceReader(d, storageMock, "test-bucket", nil)

	// Pre-setup common mock for PresignGet
	storageMock.On("PresignGet", mock.Anything, "test-bucket", mock.Anything, mock.Anything).
		Return("http://mock-url", nil).Maybe()

	// 1. Prepare Root -> A -> B
	resB := model.Resource{ID: uuid.New().String(), Name: "Res B", TypeKey: "model"}
	db.Create(&resB)
	verB := model.ResourceVersion{ID: uuid.New().String(), ResourceID: resB.ID, VersionNum: 1, SemVer: "v1.0.0", FilePath: "b.glb", State: "ACTIVE"}
	db.Create(&verB)

	resA := model.Resource{ID: uuid.New().String(), Name: "Res A", TypeKey: "model"}
	db.Create(&resA)
	verA := model.ResourceVersion{ID: uuid.New().String(), ResourceID: resA.ID, VersionNum: 1, SemVer: "v1.1.0", FilePath: "a.glb", State: "ACTIVE"}
	db.Create(&verA)
	db.Create(&model.ResourceDependency{SourceVersionID: verA.ID, TargetResourceID: resB.ID, Constraint: "latest"})

	resRoot := model.Resource{ID: uuid.New().String(), Name: "Root", TypeKey: "scenario"}
	db.Create(&resRoot)
	verRoot := model.ResourceVersion{ID: uuid.New().String(), ResourceID: resRoot.ID, VersionNum: 1, SemVer: "v2.0.0", FilePath: "root.zip", State: "ACTIVE"}
	db.Create(&verRoot)
	db.Create(&model.ResourceDependency{SourceVersionID: verRoot.ID, TargetResourceID: resA.ID, Constraint: "latest"})

	t.Run("Scan Recursive Bundle Items", func(t *testing.T) {
		bundle, err := reader.GetResourceBundle(context.Background(), verRoot.ID)
		assert.NoError(t, err)
		// Should contain Root, A, B
		assert.Len(t, bundle, 3)

		names := []string{}
		for _, item := range bundle {
			names = append(names, item["resource_name"].(string))
		}
		assert.Contains(t, names, "Root")
		assert.Contains(t, names, "Res A")
		assert.Contains(t, names, "Res B")
	})

	t.Run("Generate Bundle ZIP with Manifest", func(t *testing.T) {
		// Mock physical file access
		storageMock.On("Get", mock.Anything, "test-bucket", "root.zip").
			Return(io.NopCloser(strings.NewReader("root-data")), nil).Once()
		storageMock.On("Get", mock.Anything, "test-bucket", "a.glb").
			Return(io.NopCloser(strings.NewReader("a-data")), nil).Once()
		storageMock.On("Get", mock.Anything, "test-bucket", "b.glb").
			Return(io.NopCloser(strings.NewReader("b-data")), nil).Once()

		buf := new(bytes.Buffer)
		err := reader.DownloadBundleZip(context.Background(), verRoot.ID, buf)
		assert.NoError(t, err)

		// Verify ZIP content
		zipReader, _ := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		fileNames := []string{}
		for _, f := range zipReader.File {
			fileNames = append(fileNames, f.Name)
		}

		assert.Contains(t, fileNames, "manifest.json")
		assert.Contains(t, fileNames, "resources/scenario/Root-v2.0.0/root.zip")
		assert.Contains(t, fileNames, "resources/model/Res A-v1.1.0/a.glb")
		assert.Contains(t, fileNames, "resources/model/Res B-v1.0.0/b.glb")

		storageMock.AssertExpectations(t)
	})
}
