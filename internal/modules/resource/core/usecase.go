package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"log"
	"os"

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
	jobChan     chan processJob // 任务队列
}

type processJob struct {
	TypeKey   string
	ObjectKey string
	VersionID string
}

func NewUseCase(d *data.Data, tv *sts.TokenVendor, bucket string) *UseCase {
	uc := &UseCase{
		data:        d,
		tokenVendor: tv,
		minioConfig: bucket,
		jobChan:     make(chan processJob, 1000), // 缓冲区
	}

	// 启动固定数量的 Worker (例如 4 个并发)
	for i := 0; i < 4; i++ {
		go uc.startWorker(i)
	}

	return uc
}

func (uc *UseCase) startWorker(id int) {
	log.Printf("[Worker %d] 启动", id)
	for job := range uc.jobChan {
		uc.processResourceInternal(context.Background(), job.TypeKey, job.ObjectKey, job.VersionID)
	}
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

	// 0. 验证 MinIO 中对象是否存在
	objInfo, err := uc.tokenVendor.StatObject(ctx, uc.minioConfig, objectKey)
	if err != nil {
		log.Printf("[Confirm] 无法获取对象信息 %s: %v", objectKey, err)
		return fmt.Errorf("uploaded file not found: %w", err)
	}
	actualSize := objInfo.Size

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
			FileSize:   actualSize, // 使用 MinIO 实际大小
			MetaData:   req.ExtraMeta,
			State:      "PENDING",
		}
		if err := tx.Create(&ver).Error; err != nil {
			return err
		}

		// 发送任务到队列，而不是开启匿名 goroutine
		uc.jobChan <- processJob{
			TypeKey:   req.TypeKey,
			ObjectKey: objectKey,
			VersionID: ver.ID,
		}

		return nil
	})
}

// processResourceInternal 异步处理资源逻辑 (由 Worker 调用)
func (uc *UseCase) processResourceInternal(ctx context.Context, typeKey, objectKey, versionID string) {
	log.Printf("[Worker] 开始处理资源: %s (Type: %s)", objectKey, typeKey)

	// 1. 获取资源类型配置，检查是否有处理器
	var rt model.ResourceType
	if err := uc.data.DB.First(&rt, "type_key = ?", typeKey).Error; err != nil {
		log.Printf("[Worker] 无法获取资源类型配置: %v", typeKey)
		return
	}

	// 更新状态为 PROCESSING
	uc.data.DB.Model(&model.ResourceVersion{}).Where("id = ?", versionID).Update("state", "PROCESSING")

	var finalMeta map[string]any

	// 2. 如果有处理器，则执行处理
	if rt.ProcessorCmd != "" {
		tempDir, err := os.MkdirTemp("", "simhub-proc-*")
		if err != nil {
			log.Printf("[Worker] 创建临时目录失败: %v", err)
			return
		}
		defer os.RemoveAll(tempDir)

		// 3. 执行外部处理器指令
		// 需求变更：移除本地脚本执行，改为消息队列模式
		// TODO: 后续集成消息队列 (如 Kafka/RabbitMQ) 发送处理事件
		log.Printf("[Worker] 待发送处理消息至 MQ: Type=%s, Key=%s", rt.TypeKey, objectKey)

		// 模拟异步处理耗时
		time.Sleep(500 * time.Millisecond)

		// 暂时只做简单的元数据填充
		finalMeta = map[string]any{
			"processed_by": "simhub-core-mq-pending",
			"status":       "queued",
		}
	}

	// 3. 更新数据库并持久化 Sidecar 元数据到存储
	err := uc.data.DB.Transaction(func(tx *gorm.DB) error {
		var ver model.ResourceVersion
		if err := tx.First(&ver, "id = ?", versionID).Error; err != nil {
			return err
		}

		if ver.MetaData == nil {
			ver.MetaData = make(map[string]any)
		}
		for k, v := range finalMeta {
			ver.MetaData[k] = v
		}
		ver.State = "ACTIVE"
		if err := tx.Save(&ver).Error; err != nil {
			return err
		}

		// --- 工程级改进：写入 Metadata Sidecar ---
		// 存储位置: resources/{type}/{res_id}/{filename}.meta.json
		sidecarKey := objectKey + ".meta.json"
		sidecarData := map[string]any{
			"resource_id": ver.ResourceID,
			"version_id":  ver.ID,
			"type_key":    typeKey,
			"metadata":    ver.MetaData,
			"synced_at":   time.Now().Format(time.RFC3339),
		}
		if err := uc.tokenVendor.PutObjectJSON(ctx, uc.minioConfig, sidecarKey, sidecarData); err != nil {
			log.Printf("[Worker] 写入 Sidecar 失败 (不影响主业务): %v", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("[Worker] 数据库更新失败: %v", err)
	} else {
		log.Printf("[Worker] 处理完成: %s", objectKey)
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
	return uc.data.DB.Model(&model.Resource{}).Where("id = ?", id).Select("Tags").Updates(model.Resource{Tags: tags}).Error
}

// SyncFromStorage 从存储扫描并同步资源到数据库
func (uc *UseCase) SyncFromStorage(ctx context.Context) (int, error) {
	bucketName := uc.minioConfig
	// 1. 列出所有对象
	// 期望路径格式: resources/{type_key}/{resource_id}/{filename}
	objectCh := uc.tokenVendor.ListObjects(ctx, bucketName, "resources/")

	syncedCount := 0
	for object := range objectCh {
		if object.Err != nil {
			return syncedCount, object.Err
		}

		// 解析路径
		slashParts := strings.Split(object.Key, "/")
		if len(slashParts) < 4 {
			continue // 路径格式不对
		}

		typeKey := slashParts[1]
		resourceID := slashParts[2]
		fileName := slashParts[3]

		// 2. 检查数据库是否已存在该版本
		var exists int64
		uc.data.DB.Model(&model.ResourceVersion{}).Where("file_path = ?", object.Key).Count(&exists)
		if exists > 0 {
			continue
		}

		// 3. 尝试恢复资源主表
		var res model.Resource
		if err := uc.data.DB.First(&res, "id = ?", resourceID).Error; err != nil {
			// 如果主表不存在，创建它
			res = model.Resource{
				ID:      resourceID,
				TypeKey: typeKey,
				Name:    fileName, // 默认使用文件名作为资源名
				OwnerID: "system-sync",
			}
			if err := uc.data.DB.Create(&res).Error; err != nil {
				log.Printf("[Sync] 无法创建资源主表: %v", err)
				continue
			}
		}

		// 4. 创建版本记录
		ver := model.ResourceVersion{
			ResourceID: resourceID,
			VersionNum: 1, // 简单处理，同步默认为 v1
			FileSize:   object.Size,
			FilePath:   object.Key,
			State:      "PENDING",
			MetaData:   map[string]any{"source": "storage_sync"},
		}

		if err := uc.data.DB.Create(&ver).Error; err != nil {
			log.Printf("[Sync] 无法创建版本记录: %v", err)
			continue
		}

		// 5. 触发异步处理器（重新提取元数据和分类）
		uc.jobChan <- processJob{
			TypeKey:   typeKey,
			ObjectKey: object.Key,
			VersionID: ver.ID,
		}
		syncedCount++
	}

	return syncedCount, nil
}

// DeleteResource 删除资源 (软删除)
// DeleteResource 删除资源 (物理删除 + 存储清理)
func (uc *UseCase) DeleteResource(ctx context.Context, id string) error {
	// 1. 获取资源信息
	var res model.Resource
	if err := uc.data.DB.First(&res, "id = ?", id).Error; err != nil {
		return err
	}

	// 2. 获取所有版本
	var versions []model.ResourceVersion
	if err := uc.data.DB.Find(&versions, "resource_id = ?", id).Error; err != nil {
		return err
	}

	// 3. 删除 MinIO 中的文件 (包括元数据 Sidecar)
	for _, v := range versions {
		// 删除主文件
		if err := uc.tokenVendor.RemoveObject(ctx, uc.minioConfig, v.FilePath); err != nil {
			log.Printf("[Delete] 无法删除 MinIO 文件 %s: %v", v.FilePath, err)
			continue
		}
		// 删除 Sidecar 元数据文件
		sidecarKey := v.FilePath + ".meta.json"
		if err := uc.tokenVendor.RemoveObject(ctx, uc.minioConfig, sidecarKey); err != nil {
			log.Printf("[Delete] 无法删除 Sidecar %s: %v", sidecarKey, err)
			continue
		}
	}

	// 4. 数据库级联删除
	return uc.data.DB.Transaction(func(tx *gorm.DB) error {
		// 删除所有版本记录
		if err := tx.Delete(&model.ResourceVersion{}, "resource_id = ?", id).Error; err != nil {
			return err
		}
		// 删除资源主表记录
		if err := tx.Delete(&model.Resource{}, "id = ?", id).Error; err != nil {
			return err
		}
		return nil
	})
}
