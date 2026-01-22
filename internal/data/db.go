package data

import (
	"fmt"
	"log"

	"github.com/liny/sim-hub/internal/conf"
	"github.com/liny/sim-hub/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Data struct {
	DB *gorm.DB
}

func NewData(c *conf.Data) (*Data, func(), error) {
	var dialector gorm.Dialector

	switch c.Database.Driver {
	case "mysql":
		dialector = mysql.Open(c.Database.Source)
	case "postgres":
		dialector = postgres.Open(c.Database.Source)
	case "sqlite", "": // Default to sqlite
		dsn := "simhub.db"
		if c.Database.Source != "" {
			dsn = c.Database.Source
		}
		dialector = sqlite.Open(dsn)
	default:
		return nil, nil, fmt.Errorf("unsupported database driver: %s", c.Database.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		// DisableForeignKeyConstraintWhenMigrating: true, // Optional: useful for SQLite if strict
	})
	if err != nil {
		return nil, nil, err
	}

	// Auto Migrate
	if err := db.AutoMigrate(
		&model.ResourceType{},
		&model.Resource{},
		&model.ResourceVersion{},
	); err != nil {
		return nil, nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Seed basic types if empty
	seedBasicTypes(db)

	cleanup := func() {
		log.Println("closing the data resources")
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return &Data{DB: db}, cleanup, nil
}

func seedBasicTypes(db *gorm.DB) {
	var count int64
	db.Model(&model.ResourceType{}).Count(&count)
	if count == 0 {
		types := []model.ResourceType{
			{
				TypeKey:     "map_terrain",
				TypeName:    "Terrain Map (TIF)",
				SchemaDef:   []byte(`{"type": "object", "properties": {"resolution": {"type": "string"}}}`),
				ViewerConf:  []byte(`{"component": "CesiumViewer", "mode": "2D"}`),
				ProcessConf: []byte(`{"pipeline": ["gdal_retile"]}`),
			},
			{
				TypeKey:     "model_glb",
				TypeName:    "3D Model (GLB)",
				SchemaDef:   []byte(`{"type": "object", "properties": {"poly_count": {"type": "integer"}}}`),
				ViewerConf:  []byte(`{"component": "ThreeViewer"}`),
				ProcessConf: []byte(`{"pipeline": ["model_optimizer"]}`),
			},
		}
		db.Create(&types)
		log.Println("Seeded basic resource types")
	}
}
