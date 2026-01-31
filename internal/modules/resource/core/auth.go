package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"sim-hub/internal/data"
	"sim-hub/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthManager struct {
	data *data.Data
}

func (m *AuthManager) DB() *gorm.DB {
	return m.data.DB
}

func NewAuthManager(d *data.Data) *AuthManager {
	return &AuthManager{data: d}
}

// CreateAccessTokenResponse 返回给用户的结果，包含明文 Token
type CreateAccessTokenResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	RawToken  string    `json:"token"` // 仅在此处可见
	CreatedAt time.Time `json:"created_at"`
}

// CreateAccessToken 为用户生成一个新令牌
func (m *AuthManager) CreateAccessToken(ctx context.Context, userID string, name string, expireInDays int) (*CreateAccessTokenResponse, error) {
	// 1. 生成加密随机令牌
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	rawToken := hex.EncodeToString(b)

	// 2. 计算哈希值用于存储
	tokenHash := m.hashToken(rawToken)

	// 3. 构造数据库记录
	var expiresAt *time.Time
	if expireInDays > 0 {
		t := time.Now().AddDate(0, 0, expireInDays)
		expiresAt = &t
	}

	at := model.AccessToken{
		UserID:    userID,
		Name:      name,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}

	if err := m.data.DB.Create(&at).Error; err != nil {
		return nil, err
	}

	return &CreateAccessTokenResponse{
		ID:        at.ID,
		Name:      at.Name,
		RawToken:  "shp_" + rawToken, // 增加前缀以识别这是 SimHub 令牌
		CreatedAt: at.CreatedAt,
	}, nil
}

// RevokeAccessToken 撤销令牌
func (m *AuthManager) RevokeAccessToken(ctx context.Context, userID string, tokenID string) error {
	return m.data.DB.Delete(&model.AccessToken{}, "id = ? AND user_id = ?", tokenID, userID).Error
}

// ListAccessTokens 列出用户的所有令牌 (排除系统生成的会话令牌)
func (m *AuthManager) ListAccessTokens(ctx context.Context, userID string) ([]model.AccessToken, error) {
	var tokens []model.AccessToken
	// 排除名称为 "SimHub Web Session" 或以 "shp_" 为前缀的会话令牌 (如果有特殊标记的话)
	// 这里简单排除名称
	err := m.data.DB.Where("user_id = ? AND name != ?", userID, "SimHub Web Session").Order("created_at desc").Find(&tokens).Error
	return tokens, err
}

// VerifyToken 验证令牌有效性并返回 UserID
func (m *AuthManager) VerifyToken(ctx context.Context, rawToken string) (string, error) {
	// 去除可能的前缀
	if len(rawToken) > 4 && rawToken[:4] == "shp_" {
		rawToken = rawToken[4:]
	}

	hash := m.hashToken(rawToken)

	var at model.AccessToken
	err := m.data.DB.Where("token_hash = ?", hash).First(&at).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("invalid token")
		}
		return "", err
	}

	// 检查是否过期
	if at.ExpiresAt != nil && at.ExpiresAt.Before(time.Now()) {
		return "", fmt.Errorf("token expired")
	}

	// 异步更新最后使用时间
	go func() {
		now := time.Now()
		m.data.DB.Model(&model.AccessToken{}).Where("id = ?", at.ID).Update("last_used_at", &now)
	}()

	return at.UserID, nil
}

func (m *AuthManager) hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

// seedRoles 预埋系统基础角色与权限
func (m *AuthManager) seedRoles(_ context.Context) error {
	roles := []model.Role{
		{
			Name:        "管理员",
			Key:         "admin",
			Permissions: []string{"*"},
		},
		{
			Name: "操作员",
			Key:  "operator",
			Permissions: []string{
				"resource:list", "resource:create", "resource:update",
				"node:shell",
			},
		},
		{
			Name: "访客",
			Key:  "viewer",
			Permissions: []string{
				"resource:list",
			},
		},
	}

	for _, r := range roles {
		var existing model.Role
		if err := m.data.DB.Where("key = ?", r.Key).First(&existing).Error; err == nil {
			// 角色已存在，确保权限列表是最新的
			existing.Permissions = r.Permissions
			m.data.DB.Save(&existing)
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			// 角色不存在，创建
			if err := m.data.DB.Create(&r).Error; err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}

// EnsureAdminUser 确保管理员账户存在
func (m *AuthManager) EnsureAdminUser(ctx context.Context, defaultPwd string) error {
	// 1. 先确保角色已播种
	if err := m.seedRoles(ctx); err != nil {
		return err
	}

	// 2. 查找角色 ID
	var adminRole model.Role
	if err := m.data.DB.Where("key = ?", "admin").First(&adminRole).Error; err != nil {
		return fmt.Errorf("admin role not found: %w", err)
	}

	var existing model.User
	if err := m.data.DB.Where("username = ?", "admin").First(&existing).Error; err == nil {
		// 确保角色 ID 正确
		if existing.RoleID != adminRole.ID {
			m.data.DB.Model(&existing).Update("role_id", adminRole.ID)
		}
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := model.User{
		Username:     "admin",
		PasswordHash: string(hash),
		RoleID:       adminRole.ID,
	}
	return m.data.DB.Create(&admin).Error
}

// Login 用户登录，返回 AccessToken
func (m *AuthManager) Login(ctx context.Context, username, password string) (*CreateAccessTokenResponse, error) {
	var user model.User
	if err := m.data.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// 登录成功，生成一个 Web Session Token (本质上也是 AccessToken，但用途不同)
	// 有效期默认 7 天
	return m.CreateAccessToken(ctx, user.ID, "SimHub Web Session", 7)
}

// GetUserWithRole 获取用户及其关联的角色权限
func (m *AuthManager) GetUserWithRole(ctx context.Context, userID string) (*model.User, error) {
	var user model.User
	if err := m.data.DB.Preload("Role").Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}
