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

// NewData 初始化数据库连接并执行迁移
func NewData(c *conf.Data) (*Data, func(), error) {
	var dialector gorm.Dialector

	switch c.Database.Driver {
	case "mysql":
		dialector = mysql.Open(c.Database.Source)
	case "postgres":
		dialector = postgres.Open(c.Database.Source)
	case "sqlite", "": // 默认为 sqlite
		dsn := "simhub.db"
		if c.Database.Source != "" {
			dsn = c.Database.Source
		}
		dialector = sqlite.Open(dsn)
	default:
		return nil, nil, fmt.Errorf("不支持的数据库驱动: %s", c.Database.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, nil, err
	}

	// 执行自动迁移，同步数据库表结构
	if err := db.AutoMigrate(
		&model.ResourceType{},
		&model.Resource{},
		&model.ResourceVersion{},
	); err != nil {
		return nil, nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 若表为空，则注入基础类型数据
	seedBasicTypes(db)

	cleanup := func() {
		log.Println("正在关闭数据资源连接")
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return &Data{DB: db}, cleanup, nil
}

// seedBasicTypes 注入基础资源类型定义
func seedBasicTypes(db *gorm.DB) {
	var count int64
	db.Model(&model.ResourceType{}).Count(&count)
	if count == 0 {
		types := []model.ResourceType{
			{
				TypeKey:     "map_terrain",
				TypeName:    "地形图 (TIF)",
				SchemaDef:   []byte(`{"type": "object", "properties": {"resolution": {"type": "string"}}}`),
				ViewerConf:  []byte(`{"component": "CesiumViewer", "mode": "2D"}`),
				ProcessConf: []byte(`{"pipeline": ["gdal_retile"]}`),
			},
			{
				TypeKey:     "model_glb",
				TypeName:    "3D 模型 (GLB)",
				SchemaDef:   []byte(`{"type": "object", "properties": {"poly_count": {"type": "integer"}}}`),
				ViewerConf:  []byte(`{"component": "ThreeViewer"}`),
				ProcessConf: []byte(`{"pipeline": ["model_optimizer"]}`),
			},
		}
		db.Create(&types)
		log.Println("已注入基础资源类型")
	}
}
