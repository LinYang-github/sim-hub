package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role 角色权限模型
type Role struct {
	ID          string         `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Name        string         `gorm:"type:varchar(50);not null" json:"name"`
	Key         string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"key"` // e.g. admin, operator, viewer
	Permissions []string       `gorm:"serializer:json" json:"permissions"`               // 权限标识列表, e.g. ["resource:list", "resource:create"]
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (r *Role) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return
}

// User 用户系统
type User struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Username     string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"` // 不在 JSON 中返回
	RoleID       string    `gorm:"type:varchar(36);index" json:"role_id"`
	Role         Role      `gorm:"foreignKey:RoleID" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return
}

// AccessToken 个人访问令牌 (PAT)
// ... (AccessToken struct remains same)
type AccessToken struct {
	ID         string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID     string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	User       User       `gorm:"foreignKey:UserID" json:"-"`
	Name       string     `gorm:"type:varchar(100);not null" json:"name"` // 令牌备注名称
	TokenHash  string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (at *AccessToken) BeforeCreate(tx *gorm.DB) (err error) {
	if at.ID == "" {
		at.ID = uuid.New().String()
	}
	return
}
