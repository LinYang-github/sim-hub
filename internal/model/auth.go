package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User 用户系统
type User struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Username     string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"` // 不在 JSON 中返回
	Role         string    `gorm:"type:varchar(20);default:'user'" json:"role"`
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
