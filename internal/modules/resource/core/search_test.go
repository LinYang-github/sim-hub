package core

import (
	"context"
	"testing"

	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSearchLogicDepth(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&model.Resource{}, &model.ResourceVersion{})
	d := &data.Data{DB: db}

	reader := NewResourceReader(d, nil, "test-bucket")

	// Seed data
	db.Create(&model.Resource{Name: "Model A", TypeKey: "model", Tags: []string{"tag1", "common"}})
	db.Create(&model.Resource{Name: "Texture B", TypeKey: "texture", Tags: []string{"tag2", "common"}})
	db.Create(&model.Resource{Name: "Scenario C", TypeKey: "scenario", Tags: []string{"tag3"}})

	t.Run("Search by Name Keyword", func(t *testing.T) {
		res, total, err := reader.ListResources(context.Background(), "", "", "", "", "Model", 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.NotEmpty(t, res)
		assert.Equal(t, "Model A", res[0].Name)
	})

	t.Run("Search by Tag Keyword", func(t *testing.T) {
		res, total, err := reader.ListResources(context.Background(), "", "", "", "", "common", 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, res, 2)
	})

	t.Run("Filter by TypeKey", func(t *testing.T) {
		res, total, err := reader.ListResources(context.Background(), "scenario", "", "", "", "", 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "scenario", res[0].TypeKey)
	})

	t.Run("Search Empty Results", func(t *testing.T) {
		_, total, err := reader.ListResources(context.Background(), "", "", "", "", "nonexistent", 1, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), total)
	})
}
