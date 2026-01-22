package core

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/model"
	"github.com/liny/sim-hub/pkg/sts"
	"gorm.io/gorm"
)

type UseCase struct {
	data        *data.Data
	tokenVendor *sts.TokenVendor
	minioConfig string
}

func NewUseCase(d *data.Data, tv *sts.TokenVendor, bucket string) *UseCase {
	return &UseCase{data: d, tokenVendor: tv, minioConfig: bucket}
}

// DTOs 数据传输对象
type ApplyUploadTokenRequest struct {
	ResourceType string `json:"resource_type"`
	Checksum     string `json:"checksum"`
	Size         int64  `json:"size"`
	Filename     string `json:"filename"`
	Mode         string `json:"mode"` // "presigned" (默认) 或 "sts"
}

type ConfirmUploadRequest struct {
	TicketID  string         `json:"ticket_id"`
	TypeKey   string         `json:"type_key"`
	Name      string         `json:"name"`
	OwnerID   string         `json:"owner_id"`
	Size      int64          `json:"size"`
	ExtraMeta map[string]any `json:"extra_meta"`
}

type UploadTicket struct {
	TicketID     string              `json:"ticket_id"`
	PresignedURL string              `json:"presigned_url"`
	Credentials  *sts.STSCredentials `json:"credentials,omitempty"`
	Bucket       string              `json:"bucket,omitempty"`
	ObjectKey    string              `json:"object_key,omitempty"`
}

type ResourceDTO struct {
	ID        string              `json:"id"`
	TypeKey   string              `json:"type_key"`
	Name      string              `json:"name"`
	OwnerID   string              `json:"owner_id"`
	Tags      []string            `json:"tags"`
	CreatedAt time.Time           `json:"created_at"`
	LatestVer *ResourceVersionDTO `json:"latest_version,omitempty"`
}

type ResourceVersionDTO struct {
	VersionNum  int            `json:"version_num"`
	FileSize    int64          `json:"file_size"`
	MetaData    map[string]any `json:"meta_data"`
	DownloadURL string         `json:"download_url,omitempty"`
}

// Logic Methods 业务逻辑方法

// RequestUploadToken 请求上传令牌
func (uc *UseCase) RequestUploadToken(ctx context.Context, req ApplyUploadTokenRequest) (*UploadTicket, error) {
	ticketID := uuid.New().String()
	// objectKey 格式: resources/{type}/{uuid}/{filename}
	objectKey := "resources/" + req.ResourceType + "/" + ticketID + "/" + req.Filename

	if uc.tokenVendor == nil {
		return nil, gorm.ErrInvalidDB // 或者返回自定义错误 "Storage Service Unavailable"
	}

	if req.Mode == "sts" {
		creds, err := uc.tokenVendor.GenerateUploadToken(ctx, uc.minioConfig, objectKey, time.Hour)
		if err != nil {
			return nil, err
		}
		return &UploadTicket{
			TicketID:    ticketID + "::" + objectKey,
			Credentials: creds,
			Bucket:      uc.minioConfig,
			ObjectKey:   objectKey,
		}, nil
	}

	// 默认模式: 预签名 URL
	url, err := uc.tokenVendor.GeneratePresignedUpload(ctx, uc.minioConfig, objectKey, time.Hour)
	if err != nil {
		return nil, err
	}

	return &UploadTicket{
		TicketID:     ticketID + "::" + objectKey, // 简易存储以实现无状态验证（生产环境建议使用 Redis）
		PresignedURL: url,
	}, nil
}

// ConfirmUpload 确认上传完成
func (uc *UseCase) ConfirmUpload(ctx context.Context, req ConfirmUploadRequest) error {
	objectKey := ""
	if len(req.TicketID) > 36 {
		objectKey = req.TicketID[37:]
	}

	return uc.data.DB.Transaction(func(tx *gorm.DB) error {
		res := model.Resource{
			TypeKey: req.TypeKey,
			Name:    req.Name,
			OwnerID: req.OwnerID,
		}
		if err := tx.Create(&res).Error; err != nil {
			return err
		}

		ver := model.ResourceVersion{
			ResourceID: res.ID,
			VersionNum: 1,
			FilePath:   objectKey,
			FileSize:   req.Size,
			MetaData:   req.ExtraMeta,
			State:      "ACTIVE",
		}
		if err := tx.Create(&ver).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetResource 获取资源详情
func (uc *UseCase) GetResource(ctx context.Context, id string) (*ResourceDTO, error) {
	var r model.Resource
	if err := uc.data.DB.First(&r, "id = ?", id).Error; err != nil {
		return nil, err
	}

	var v model.ResourceVersion
	if err := uc.data.DB.Order("version_num desc").First(&v, "resource_id = ?", id).Error; err != nil {
		return nil, err
	}

	url, err := uc.tokenVendor.GenerateDownloadURL(ctx, uc.minioConfig, v.FilePath, time.Hour)
	if err != nil {
		return nil, err
	}

	return &ResourceDTO{
		ID:        r.ID,
		TypeKey:   r.TypeKey,
		Name:      r.Name,
		OwnerID:   r.OwnerID,
		Tags:      r.Tags,
		CreatedAt: r.CreatedAt,
		LatestVer: &ResourceVersionDTO{
			VersionNum:  v.VersionNum,
			FileSize:    v.FileSize,
			MetaData:    v.MetaData,
			DownloadURL: url,
		},
	}, nil
}

// ListResources 列出资源
func (uc *UseCase) ListResources(ctx context.Context, typeKey string, page, size int) ([]*ResourceDTO, int64, error) {
	var resources []model.Resource
	var total int64
	offset := (page - 1) * size

	query := uc.data.DB.Model(&model.Resource{})
	if typeKey != "" {
		query = query.Where("type_key = ?", typeKey)
	}

	if err := query.Count(&total).Limit(size).Offset(offset).Order("created_at desc").Find(&resources).Error; err != nil {
		return nil, 0, err
	}

	cw := make([]*ResourceDTO, 0, len(resources))
	for _, r := range resources {
		cw = append(cw, &ResourceDTO{
			ID:        r.ID,
			TypeKey:   r.TypeKey,
			Name:      r.Name,
			OwnerID:   r.OwnerID,
			Tags:      r.Tags,
			CreatedAt: r.CreatedAt,
		})
	}
	return cw, total, nil
}
