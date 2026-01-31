package resource

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"sim-hub/internal/conf"
	"sim-hub/internal/data"
	"sim-hub/internal/model"
	"sim-hub/internal/modules/resource/core"
	"sim-hub/internal/modules/resource/core/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAPIEnv(t *testing.T) (*gin.Engine, *mocks.MockBlobStore) {
	gin.SetMode(gin.TestMode)
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.ResourceType{})
	d := &data.Data{DB: db}

	// Mock components
	mockStore := new(mocks.MockBlobStore)
	mockSTS := new(mocks.MockSTSProvider)

	// Minimal resource types for testing
	resTypes := []conf.ResourceType{
		{TypeKey: "scenario", TypeName: "Scenario"},
	}

	m := NewModule(d, mockStore, mockSTS, "test-bucket", nil, "api", "http://test", nil, resTypes)

	r := gin.New()
	api := r.Group("/api/v1")
	m.RegisterRoutes(api)

	return r, mockStore
}

func TestApplyUploadTokenAPI(t *testing.T) {
	r, mockStore := setupAPIEnv(t)

	t.Run("Successfully get upload token", func(t *testing.T) {
		mockStore.On("PresignPut", mock.Anything, "test-bucket", mock.Anything, mock.Anything).
			Return("http://mock-url", nil).Once()

		reqBody := core.ApplyUploadTokenRequest{
			ResourceType: "scenario",
			Filename:     "test.zip",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/integration/upload/token", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "http://mock-url", resp["presigned_url"])
		mockStore.AssertExpectations(t)
	})

	t.Run("Invalid JSON should return 400", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/integration/upload/token", bytes.NewBufferString("invalid-json"))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestResourceTypesAPI(t *testing.T) {
	r, _ := setupAPIEnv(t)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/resource-types", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)

	assert.NotEmpty(t, resp)
	assert.Equal(t, "scenario", resp[0]["type_key"])
}
