package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ResourceType 资源类型定义
type ResourceType struct {
	TypeKey         string         `gorm:"primaryKey;type:varchar(50)" json:"type_key"`
	TypeName        string         `gorm:"type:varchar(100);not null" json:"type_name"`
	SchemaDef       map[string]any `gorm:"serializer:json" json:"schema_def"`                    // 前端表单定义的 JSON Schema
	CategoryMode    string         `gorm:"type:varchar(20);default:'flat'" json:"category_mode"` // "flat" 或 "tree"
	IntegrationMode string         `gorm:"type:varchar(20);default:'internal'" json:"integration_mode"`
	UploadMode      string         `gorm:"type:varchar(50)" json:"upload_mode"`
	ProcessConf     map[string]any `gorm:"serializer:json" json:"process_conf"`
	MetaData        map[string]any `gorm:"serializer:json" json:"meta_data"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// Category 资源分类（虚拟文件夹）
type Category struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TypeKey   string    `gorm:"type:varchar(50);not null;index" json:"type_key"` // 属于哪种资源类型的分类
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	ParentID  string    `gorm:"type:varchar(36);index" json:"parent_id"` // 支持层级目录
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (c *Category) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return
}

// Resource 资源主表
type Resource struct {
	ID              string       `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TypeKey         string       `gorm:"type:varchar(50);not null;index" json:"type_key"`
	ResourceType    ResourceType `gorm:"foreignKey:TypeKey;references:TypeKey" json:"resource_type,omitempty"`
	CategoryID      string       `gorm:"type:varchar(36);index" json:"category_id"` // 所属分类
	Category        *Category    `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Name            string       `gorm:"type:varchar(200);not null" json:"name"`
	OwnerID         string       `gorm:"type:varchar(50);index" json:"owner_id"`
	Scope           string       `gorm:"type:varchar(20);default:'PRIVATE';index" json:"scope"` // PRIVATE, PUBLIC
	Tags            []string     `gorm:"serializer:json" json:"tags"`                           // SQLite/MySQL doesn't support array type natively, use JSON serializer
	IsDeleted       bool         `gorm:"default:false" json:"is_deleted"`
	LatestVersionID string       `gorm:"type:varchar(36)" json:"latest_version_id"` // 指向最新发布的版本
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
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
	ResourceID string         `gorm:"type:varchar(36);not null;index:idx_res_ver_num,unique;index:idx_res_ver_sem,unique" json:"resource_id"`
	Resource   Resource       `gorm:"foreignKey:ResourceID" json:"resource,omitempty"`
	VersionNum int            `gorm:"not null;index:idx_res_ver_num,unique" json:"version_num"`
	SemVer     string         `gorm:"type:varchar(50);index:idx_res_ver_sem,unique" json:"semver"` // 语义化版本，如 v1.0.1
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

// ResourceDependency 资源依赖表
type ResourceDependency struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	SourceVersionID  string    `gorm:"type:varchar(36);not null;index" json:"source_version_id"`  // 谁产生了依赖
	TargetResourceID string    `gorm:"type:varchar(36);not null;index" json:"target_resource_id"` // 依赖了谁
	Constraint       string    `gorm:"type:varchar(100)" json:"version_constraint"`               // 版本约束，如 "^1.0.0" 或 "latest"
	TargetVersionID  string    `gorm:"type:varchar(36)" json:"target_version_id"`                 // 锁定到的具体版本（可选）
	CreatedAt        time.Time `json:"created_at"`
}
