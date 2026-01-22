package core

import (
	"context"
	"time"

	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"

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
	TicketID   string         `json:"ticket_id"`
	TypeKey    string         `json:"type_key"`
	CategoryID string         `json:"category_id"` // 新增：所属分类 ID
	Name       string         `json:"name"`
	OwnerID    string         `json:"owner_id"`
	Tags       []string       `json:"tags"` // 新增：资源标签
	Size       int64          `json:"size"`
	ExtraMeta  map[string]any `json:"extra_meta"`
}

type UpdateResourceTagsRequest struct {
	Tags []string `json:"tags"`
}

type CategoryDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
}

type CreateCategoryRequest struct {
	TypeKey  string `json:"type_key"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
}

type UploadTicket struct {
	TicketID     string              `json:"ticket_id"`
	PresignedURL string              `json:"presigned_url"`
	Credentials  *sts.STSCredentials `json:"credentials,omitempty"`
	Bucket       string              `json:"bucket,omitempty"`
	ObjectKey    string              `json:"object_key,omitempty"`
}

type ResourceDTO struct {
	ID         string              `json:"id"`
	TypeKey    string              `json:"type_key"`
	CategoryID string              `json:"category_id,omitempty"`
	Name       string              `json:"name"`
	OwnerID    string              `json:"owner_id"`
	Tags       []string            `json:"tags"`
	CreatedAt  time.Time           `json:"created_at"`
	LatestVer  *ResourceVersionDTO `json:"latest_version,omitempty"`
}

type ResourceVersionDTO struct {
	VersionNum  int            `json:"version_num"`
	FileSize    int64          `json:"file_size"`
	MetaData    map[string]any `json:"meta_data"`
	State       string         `json:"state"`
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
	if len(req.TicketID) > 38 {
		objectKey = req.TicketID[38:]
	}

	return uc.data.DB.Transaction(func(tx *gorm.DB) error {
		res := model.Resource{
			TypeKey:    req.TypeKey,
			CategoryID: req.CategoryID,
			Name:       req.Name,
			OwnerID:    req.OwnerID,
			Tags:       req.Tags,
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
			State:      "PENDING",
		}
		if err := tx.Create(&ver).Error; err != nil {
			return err
		}

		// 触发异步处理
		go uc.asyncProcessResource(context.Background(), req.TypeKey, objectKey, ver.ID)

		return nil
	})
}

// asyncProcessResource 异步处理资源逻辑
func (uc *UseCase) asyncProcessResource(ctx context.Context, typeKey, objectKey, versionID string) {
	log.Printf("[Processor] 开始处理资源: %s (Type: %s)", objectKey, typeKey)

	// 1. 获取资源类型配置，检查是否有处理器
	var rt model.ResourceType
	if err := uc.data.DB.First(&rt, "type_key = ?", typeKey).Error; err != nil {
		log.Printf("[Processor] 无法获取资源类型配置: %v", err)
		return
	}

	// 更新状态为 PROCESSING
	uc.data.DB.Model(&model.ResourceVersion{}).Where("id = ?", versionID).Update("state", "PROCESSING")

	// 如果没有处理器指令，直接设为 ACTIVE
	if rt.ProcessorCmd == "" {
		log.Printf("[Processor] 无需处理器，直接设为 ACTIVE")
		uc.data.DB.Model(&model.ResourceVersion{}).Where("id = ?", versionID).Update("state", "ACTIVE")
		return
	}

	// 2. 准备临时工作目录与文件
	tempDir, err := os.MkdirTemp("", "simhub-proc-*")
	if err != nil {
		log.Printf("[Processor] 创建临时目录失败: %v", err)
		return
	}
	defer os.RemoveAll(tempDir)

	localFile := filepath.Join(tempDir, filepath.Base(objectKey))
	err = uc.tokenVendor.FGetObject(ctx, uc.minioConfig, objectKey, localFile)
	if err != nil {
		log.Printf("[Processor] 下载文件失败: %v", err)
		uc.data.DB.Model(&model.ResourceVersion{}).Where("id = ?", versionID).Update("state", "FAILED")
		return
	}

	// 3. 执行外部处理器指令
	// 契约：处理器通过环境变量或参数接收文件路径，通过 stdout 输出 JSON
	cmd := exec.Command(rt.ProcessorCmd, "--file", localFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[Processor] 驱动执行失败: %v, Output: %s", err, string(output))
		uc.data.DB.Model(&model.ResourceVersion{}).Where("id = ?", versionID).Update("state", "FAILED")
		return
	}

	// 4. 解析输出结果并回填元数据
	var result struct {
		Status   string         `json:"status"`
		Metadata map[string]any `json:"metadata"`
		Error    string         `json:"error"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		log.Printf("[Processor] 无法解析驱动输出 (预期JSON): %v, 原输出: %s", err, string(output))
		uc.data.DB.Model(&model.ResourceVersion{}).Where("id = ?", versionID).Update("state", "FAILED")
		return
	}

	if result.Status == "failed" {
		log.Printf("[Processor] 驱动反馈失败: %s", result.Error)
		uc.data.DB.Model(&model.ResourceVersion{}).Where("id = ?", versionID).Update("state", "FAILED")
		return
	}

	// 合并元数据
	err = uc.data.DB.Transaction(func(tx *gorm.DB) error {
		var ver model.ResourceVersion
		if err := tx.First(&ver, "id = ?", versionID).Error; err != nil {
			return err
		}

		// 合并原始 MetaData 和驱动解析出的 Metadata
		if ver.MetaData == nil {
			ver.MetaData = make(map[string]any)
		}
		for k, v := range result.Metadata {
			ver.MetaData[k] = v
		}

		ver.State = "ACTIVE"
		return tx.Save(&ver).Error
	})

	if err != nil {
		log.Printf("[Processor] 更新数据库记录失败: %v", err)
	} else {
		log.Printf("[Processor] 处理完成，资源已激活: %s", objectKey)
	}
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
		ID:         r.ID,
		TypeKey:    r.TypeKey,
		CategoryID: r.CategoryID,
		Name:       r.Name,
		OwnerID:    r.OwnerID,
		Tags:       r.Tags,
		CreatedAt:  r.CreatedAt,
		LatestVer: &ResourceVersionDTO{
			VersionNum:  v.VersionNum,
			FileSize:    v.FileSize,
			MetaData:    v.MetaData,
			State:       v.State,
			DownloadURL: url,
		},
	}, nil
}

// ListResources 列出资源
func (uc *UseCase) ListResources(ctx context.Context, typeKey string, categoryID string, page, size int) ([]*ResourceDTO, int64, error) {
	var resources []model.Resource
	var total int64
	offset := (page - 1) * size

	query := uc.data.DB.Model(&model.Resource{})
	if typeKey != "" {
		query = query.Where("type_key = ?", typeKey)
	}
	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	if err := query.Count(&total).Limit(size).Offset(offset).Order("created_at desc").Find(&resources).Error; err != nil {
		return nil, 0, err
	}

	cw := make([]*ResourceDTO, 0, len(resources))
	for _, r := range resources {
		// 获取最新版本以显示状态
		var v model.ResourceVersion
		uc.data.DB.Order("version_num desc").First(&v, "resource_id = ?", r.ID)

		cw = append(cw, &ResourceDTO{
			ID:         r.ID,
			TypeKey:    r.TypeKey,
			CategoryID: r.CategoryID,
			Name:       r.Name,
			OwnerID:    r.OwnerID,
			Tags:       r.Tags,
			CreatedAt:  r.CreatedAt,
			LatestVer: &ResourceVersionDTO{
				VersionNum: v.VersionNum,
				State:      v.State,
				MetaData:   v.MetaData,
			},
		})
	}
	return cw, total, nil
}

// CreateCategory 创建分类
func (uc *UseCase) CreateCategory(ctx context.Context, req CreateCategoryRequest) (*CategoryDTO, error) {
	cat := model.Category{
		TypeKey:  req.TypeKey,
		Name:     req.Name,
		ParentID: req.ParentID,
	}
	if err := uc.data.DB.Create(&cat).Error; err != nil {
		return nil, err
	}
	return &CategoryDTO{ID: cat.ID, Name: cat.Name, ParentID: cat.ParentID}, nil
}

// ListCategories 列出分类
func (uc *UseCase) ListCategories(ctx context.Context, typeKey string) ([]*CategoryDTO, error) {
	var cats []model.Category
	if err := uc.data.DB.Where("type_key = ?", typeKey).Find(&cats).Error; err != nil {
		return nil, err
	}

	res := make([]*CategoryDTO, 0, len(cats))
	for _, c := range cats {
		res = append(res, &CategoryDTO{ID: c.ID, Name: c.Name, ParentID: c.ParentID})
	}
	return res, nil
}

// DeleteCategory 删除分类
func (uc *UseCase) DeleteCategory(ctx context.Context, id string) error {
	return uc.data.DB.Delete(&model.Category{}, "id = ?", id).Error
}

// UpdateResourceTags 更新资源标签
func (uc *UseCase) UpdateResourceTags(ctx context.Context, id string, tags []string) error {
	return uc.data.DB.Model(&model.Resource{}).Where("id = ?", id).Update("tags", tags).Error
}
