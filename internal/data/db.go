package data

import (
	"fmt"
	"log/slog"

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
		&model.Category{},
		&model.Resource{},
		&model.ResourceVersion{},
		&model.ResourceDependency{},
	); err != nil {
		return nil, nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	// 从配置中注入基础类型数据
	seedBasicTypes(db, c.ResourceTypes)

	cleanup := func() {
		slog.Info("正在关闭数据资源连接")
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return &Data{DB: db}, cleanup, nil
}

// seedBasicTypes 从配置中注入基础资源类型定义
func seedBasicTypes(db *gorm.DB, configTypes []conf.ResourceType) {
	var count int64
	db.Model(&model.ResourceType{}).Count(&count)
	if count == 0 {
		var types []model.ResourceType
		for _, ct := range configTypes {
			types = append(types, model.ResourceType{
				TypeKey:         ct.TypeKey,
				TypeName:        ct.TypeName,
				SchemaDef:       ct.SchemaDef,
				CategoryMode:    ct.CategoryMode,
				IntegrationMode: ct.IntegrationMode,
				UploadMode:      ct.UploadMode,
				ProcessConf:     ct.ProcessConf,
				MetaData:        ct.MetaData,
			})
		}
		if len(types) > 0 {
			db.Create(&types)
			slog.Info("已从配置注入资源类型定义", "count", len(types))
		}
	}
}
