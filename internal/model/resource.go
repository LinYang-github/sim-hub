package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ResourceType 资源类型定义
type ResourceType struct {
	TypeKey      string    `gorm:"primaryKey;type:varchar(50)" json:"type_key"`
	TypeName     string    `gorm:"type:varchar(100);not null" json:"type_name"`
	SchemaDef    []byte    `gorm:"serializer:json" json:"schema_def"`      // 前端表单定义的 JSON Schema
	ViewerConf   []byte    `gorm:"serializer:json" json:"viewer_conf"`     // 前端预览组件配置
	ProcessConf  []byte    `gorm:"serializer:json" json:"process_conf"`    // 后端处理管线配置 (JSON)
	ProcessorCmd string    `gorm:"type:varchar(255)" json:"processor_cmd"` // 处理器执行指令 (如: /usr/bin/scenario-processor)
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Resource 资源主表
type Resource struct {
	ID           string       `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TypeKey      string       `gorm:"type:varchar(50);not null;index" json:"type_key"`
	ResourceType ResourceType `gorm:"foreignKey:TypeKey;references:TypeKey" json:"resource_type,omitempty"`
	Name         string       `gorm:"type:varchar(200);not null" json:"name"`
	OwnerID      string       `gorm:"type:varchar(50);index" json:"owner_id"`
	Tags         []string     `gorm:"serializer:json" json:"tags"` // SQLite/MySQL doesn't support array type natively, use JSON serializer
	IsDeleted    bool         `gorm:"default:false" json:"is_deleted"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

func (r *Resource) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return
}

// ResourceVersion 资源版本表
type ResourceVersion struct {
	ID         string         `gorm:"primaryKey;type:varchar(36)" json:"id"`
	ResourceID string         `gorm:"type:varchar(36);not null;index:idx_res_ver,unique" json:"resource_id"`
	Resource   Resource       `gorm:"foreignKey:ResourceID" json:"resource,omitempty"`
	VersionNum int            `gorm:"not null;index:idx_res_ver,unique" json:"version_num"`
	FilePath   string         `gorm:"type:varchar(500);not null" json:"file_path"`
	FileHash   string         `gorm:"type:varchar(64)" json:"file_hash"`
	FileSize   int64          `json:"file_size"`
	MetaData   map[string]any `gorm:"serializer:json" json:"meta_data"`                // 动态扩展属性
	State      string         `gorm:"type:varchar(20);default:'PENDING'" json:"state"` // PENDING, ACTIVE, ARCHIVED
	CreatedAt  time.Time      `json:"created_at"`
}

func (rv *ResourceVersion) BeforeCreate(tx *gorm.DB) (err error) {
	if rv.ID == "" {
		rv.ID = uuid.New().String()
	}
	return
}
