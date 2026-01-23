package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/model"
	"github.com/liny/sim-hub/pkg/storage"
	"gorm.io/gorm"
)

type UseCase struct {
	data        *data.Data
	store       storage.MultipartBlobStore
	stsProvider storage.SecurityTokenProvider
	minioConfig string
	jobChan     chan processJob // 任务队列 (本地模式使用)
	nats        *data.NATSClient
	role        string // "api", "worker", "combined"
	apiBaseURL  string
	handlers    map[string]string // 资源类型与处理器的映射
}

const (
	ActionProcess = "PROCESS" // 全流程处理 (执行 Processor + 写 Sidecar)
	ActionRefresh = "REFRESH" // 仅刷新元数据 (重新生成 Sidecar)
)

type processJob struct {
	Action    string
	TypeKey   string
	ObjectKey string
	VersionID string
}

func NewUseCase(d *data.Data, store storage.MultipartBlobStore, stsProvider storage.SecurityTokenProvider, bucket string, natsClient *data.NATSClient, role string, apiBaseURL string, handlers map[string]string) *UseCase {
	uc := &UseCase{
		data:        d,
		store:       store,
		stsProvider: stsProvider,
		minioConfig: bucket,
		jobChan:     make(chan processJob, 1000), // 缓冲区
		nats:        natsClient,
		role:        role,
		apiBaseURL:  apiBaseURL,
		handlers:    handlers,
	}

	// 任务消费者启动逻辑
	if role == "worker" || role == "combined" {
		if natsClient != nil && natsClient.Config.Enabled {
			// 分布式模式：启动 NATS 订阅者
			go uc.startNATSSubscriber()
		} else {
			// 本地模式：启动内部 Worker
			for i := 0; i < 4; i++ {
				go uc.startWorker(i)
			}
		}
	} else {
		slog.Info("当前节点为 API 模式，不启动本地任务执行器")
	}

	return uc
}

func (uc *UseCase) dispatchJob(job processJob) {
	// ActionRefresh 需要数据库访问，强制在本地执行 (API 节点有 DB)
	if job.Action == ActionRefresh {
		go uc.handleJob(context.Background(), job)
		return
	}

	if uc.nats != nil && uc.nats.Config.Enabled {
		if err := uc.nats.Encoded.Publish(uc.nats.Config.Subject, &job); err != nil {
			slog.Error("发送 NATS 消息失败，回退到本地队列", "error", err)
			// 如果是 API 模式且未启动 Worker，这里写入 jobChan 可能会阻塞或死锁
			// 但一般 fallback 意味着 NATS 挂了，系统降级
			uc.jobChan <- job
		}
		return
	}
	uc.jobChan <- job
}

func (uc *UseCase) startNATSSubscriber() {
	slog.Info("NATS 订阅者已启动", "subject", uc.nats.Config.Subject)
	_, err := uc.nats.Encoded.Subscribe(uc.nats.Config.Subject, func(job *processJob) {
		slog.Debug("接收到 NATS 任务", "action", job.Action, "key", job.ObjectKey)
		uc.handleJob(context.Background(), *job)
	})
	if err != nil {
		slog.Error("NATS 订阅失败", "error", err)
	}
}

func (uc *UseCase) startWorker(id int) {
	slog.Info("本地 Worker 启动", "worker_id", id)
	for job := range uc.jobChan {
		uc.handleJob(context.Background(), job)
	}
}

func (uc *UseCase) handleJob(ctx context.Context, job processJob) {
	switch job.Action {
	case ActionProcess:
		uc.processResourceInternal(ctx, job.TypeKey, job.ObjectKey, job.VersionID)
	case ActionRefresh:
		uc.syncSidecarInternal(ctx, job.ObjectKey, job.VersionID)
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

// Multipart Upload DTOs
type InitMultipartUploadRequest struct {
	ResourceType string `json:"resource_type"`
	Filename     string `json:"filename"`
}

type InitMultipartUploadResponse struct {
	TicketID  string `json:"ticket_id"`
	UploadID  string `json:"upload_id"`
	Bucket    string `json:"bucket"`
	ObjectKey string `json:"object_key"`
}

type GetPartURLRequest struct {
	TicketID   string `json:"ticket_id"`
	UploadID   string `json:"upload_id"`
	PartNumber int    `json:"part_number"`
}

type GetPartURLResponse struct {
	URL string `json:"url"`
}

type ProcessResultRequest struct {
	MetaData map[string]any `json:"meta_data"`
	State    string         `json:"state"` // ACTIVE, ERROR
	Message  string         `json:"message,omitempty"`
}

type CompleteMultipartUploadRequest struct {
	TicketID   string         `json:"ticket_id"`
	UploadID   string         `json:"upload_id"`
	Parts      []storage.Part `json:"parts"`
	TypeKey    string         `json:"type_key"`
	CategoryID string         `json:"category_id"`
	Name       string         `json:"name"`
	OwnerID    string         `json:"owner_id"`
	Tags       []string       `json:"tags"`
	ExtraMeta  map[string]any `json:"extra_meta"`
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
	TicketID     string                  `json:"ticket_id"`
	PresignedURL string                  `json:"presigned_url"`
	Credentials  *storage.STSCredentials `json:"credentials,omitempty"`
	Bucket       string                  `json:"bucket,omitempty"`
	ObjectKey    string                  `json:"object_key,omitempty"`
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

	if uc.stsProvider == nil {
		return nil, gorm.ErrInvalidDB // 或者返回自定义错误
	}

	if req.Mode == "sts" {
		creds, err := uc.stsProvider.GenerateSTSToken(ctx, uc.minioConfig, objectKey, time.Hour)
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
	url, err := uc.store.PresignPut(ctx, uc.minioConfig, objectKey, time.Hour)
	if err != nil {
		return nil, err
	}

	return &UploadTicket{
		TicketID:     ticketID + "::" + objectKey, // 简易存储以实现无状态验证（生产环境建议使用 Redis）
		PresignedURL: url,
	}, nil
}

// InitMultipartUpload 初始化分片上传
func (uc *UseCase) InitMultipartUpload(ctx context.Context, req InitMultipartUploadRequest) (*InitMultipartUploadResponse, error) {
	ticketID := uuid.New().String()
	objectKey := "resources/" + req.ResourceType + "/" + ticketID + "/" + req.Filename

	uploadID, err := uc.store.InitMultipart(ctx, uc.minioConfig, objectKey)
	if err != nil {
		slog.Error("初始化分片上传失败", "error", err, "key", objectKey)
		return nil, err
	}

	return &InitMultipartUploadResponse{
		TicketID:  ticketID + "::" + objectKey,
		UploadID:  uploadID,
		Bucket:    uc.minioConfig,
		ObjectKey: objectKey,
	}, nil
}

// GetMultipartUploadPartURL 获取分片上传的预签名 URL
func (uc *UseCase) GetMultipartUploadPartURL(ctx context.Context, req GetPartURLRequest) (*GetPartURLResponse, error) {
	objectKey := ""
	if len(req.TicketID) > 38 {
		objectKey = req.TicketID[38:]
	}

	url, err := uc.store.PresignPart(ctx, uc.minioConfig, objectKey, req.UploadID, req.PartNumber, time.Hour)
	if err != nil {
		slog.Error("生成分片上传 URL 失败", "error", err, "key", objectKey, "part", req.PartNumber)
		return nil, err
	}

	return &GetPartURLResponse{URL: url}, nil
}

// CompleteMultipartUpload 完成分片上传并注册资源
func (uc *UseCase) CompleteMultipartUpload(ctx context.Context, req CompleteMultipartUploadRequest) error {
	objectKey := ""
	if len(req.TicketID) > 38 {
		objectKey = req.TicketID[38:]
	}

	// 1. 在存储层完成分片合并
	if err := uc.store.CompleteMultipart(ctx, uc.minioConfig, objectKey, req.UploadID, req.Parts); err != nil {
		slog.Error("完成分片上传失败", "error", err, "key", objectKey, "upload_id", req.UploadID)
		return err
	}

	// 2. 获取最终对象信息（获取真实大小）
	objInfo, err := uc.store.Stat(ctx, uc.minioConfig, objectKey)
	if err != nil {
		slog.Error("无法获取合并后对象信息", "key", objectKey, "error", err)
		return fmt.Errorf("uploaded file not found after completion: %w", err)
	}

	// 3. 注册到数据库
	return uc.data.DB.Transaction(func(tx *gorm.DB) error {
		return uc.createResourceAndVersion(tx, req.TypeKey, req.CategoryID, req.Name, req.OwnerID, objectKey, objInfo.Size, req.Tags, req.ExtraMeta)
	})
}

// ConfirmUpload 确认上传完成
func (uc *UseCase) ConfirmUpload(ctx context.Context, req ConfirmUploadRequest) error {
	objectKey := ""
	if len(req.TicketID) > 38 {
		objectKey = req.TicketID[38:]
	}

	// 0. 验证 MinIO 中对象是否存在
	objInfo, err := uc.store.Stat(ctx, uc.minioConfig, objectKey)
	if err != nil {
		slog.Error("无法获取对象信息", "key", objectKey, "error", err)
		return fmt.Errorf("uploaded file not found: %w", err)
	}

	return uc.data.DB.Transaction(func(tx *gorm.DB) error {
		return uc.createResourceAndVersion(tx, req.TypeKey, req.CategoryID, req.Name, req.OwnerID, objectKey, objInfo.Size, req.Tags, req.ExtraMeta)
	})
}

// createResourceAndVersion 内部统一资源注册逻辑
func (uc *UseCase) createResourceAndVersion(tx *gorm.DB, typeKey, categoryID, name, ownerID, objectKey string, size int64, tags []string, meta map[string]any) error {
	res := model.Resource{
		TypeKey:    typeKey,
		CategoryID: categoryID,
		Name:       name,
		OwnerID:    ownerID,
		Tags:       tags,
	}
	if err := tx.Create(&res).Error; err != nil {
		return err
	}

	ver := model.ResourceVersion{
		ResourceID: res.ID,
		VersionNum: 1,
		FilePath:   objectKey,
		FileSize:   size,
		MetaData:   meta,
		State:      "PENDING",
	}
	if err := tx.Create(&ver).Error; err != nil {
		return err
	}

	// 触发异步处理
	uc.dispatchJob(processJob{
		Action:    ActionProcess,
		TypeKey:   typeKey,
		ObjectKey: objectKey,
		VersionID: ver.ID,
	})
	return nil
}

// processResourceInternal 异步处理资源逻辑 (由 Worker 调用)
func (uc *UseCase) processResourceInternal(ctx context.Context, typeKey, objectKey, versionID string) {
	slog.Debug("开始处理资源", "key", objectKey, "type", typeKey, "role", uc.role)

	// 1. 查询本地是否存在对应的处理器
	processorCmd := uc.handlers[typeKey]

	finalMeta := make(map[string]any)
	if processorCmd != "" {
		// --- 真实执行逻辑 ---
		// 1. 下载文件到本地临时目录
		ext := ""
		if parts := strings.Split(objectKey, "."); len(parts) > 1 {
			ext = "." + parts[len(parts)-1]
		}

		tempFile, err := os.CreateTemp("", "simhub-resource-*"+ext)
		if err != nil {
			slog.Error("创建临时文件失败", "error", err)
			return // 应该上报 ERROR 状态
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		// 从 MinIO 下载
		obj, err := uc.store.Get(ctx, uc.minioConfig, objectKey)
		if err != nil {
			slog.Error("下载资源文件失败", "key", objectKey, "error", err)
			return
		}

		if _, err := io.Copy(tempFile, obj); err != nil {
			obj.Close()
			slog.Error("保存临时文件失败", "error", err)
			return
		}
		obj.Close()

		slog.Info("文件已下载至本地，准备处理", "path", tempFile.Name())

		// 2. 执行外部命令
		// 格式: <cmd> <filepath>
		// 输出: JSON 格式的 metadata 到 stdout
		cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("%s '%s'", processorCmd, tempFile.Name()))
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		slog.Debug("执行外部处理器", "cmd", cmd.String())
		startTime := time.Now()
		if err := cmd.Run(); err != nil {
			slog.Error("外部处理器执行失败", "error", err, "stderr", stderr.String())
			// 上报错误状态
			uc.notifyResult(ctx, versionID, ProcessResultRequest{
				State:   "ERROR",
				Message: fmt.Sprintf("Processor failed: %v, stderr: %s", err, stderr.String()),
			})
			return
		}

		duration := time.Since(startTime)
		slog.Info("外部处理器执行完成", "duration", duration)

		// 3. 解析结果
		if err := json.Unmarshal(stdout.Bytes(), &finalMeta); err != nil {
			slog.Warn("处理器输出非 JSON 格式，忽略元数据", "output", stdout.String())
			finalMeta["raw_output"] = stdout.String()
		} else {
			// 追加系统级元数据
			finalMeta["processed_by"] = "simhub-worker"
			finalMeta["processed_at"] = time.Now().Format(time.RFC3339)
			finalMeta["processor_duration_ms"] = duration.Milliseconds()
		}
	} else {
		slog.Debug("未配置该类型的处理器，跳过计算", "type", typeKey)
		finalMeta["status"] = "skipped"
	}

	// 2. 上报结果
	err := uc.notifyResult(ctx, versionID, ProcessResultRequest{
		MetaData: finalMeta,
		State:    "ACTIVE",
	})

	if err != nil {
		slog.Error("处理结果上报失败", "error", err)
	} else {
		slog.Debug("资源处理结果已成功同步", "key", objectKey)
	}
}

// notifyResult 根据节点角色选择上报方式（直接写库或通过 HTTP API）
func (uc *UseCase) notifyResult(ctx context.Context, versionID string, req ProcessResultRequest) error {
	if uc.role == "api" || uc.role == "combined" {
		// 本地模式：直接调用内部方法写库
		return uc.ReportProcessResult(ctx, versionID, req)
	}

	// 远程 Worker 模式：通过 HTTP Callback 上报给 API 节点
	callbackURL := fmt.Sprintf("%s/api/v1/resources/%s/process-result", uc.apiBaseURL, versionID)
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", callbackURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("callback failed with status: %d", resp.StatusCode)
	}

	return nil
}

// syncSidecarInternal 仅执行元数据同步到存储 (不涉及外部 Processor)
func (uc *UseCase) syncSidecarInternal(ctx context.Context, objectKey, versionID string) {
	var ver model.ResourceVersion
	var res model.Resource
	if err := uc.data.DB.Preload("Resource").First(&ver, "id = ?", versionID).Error; err != nil {
		slog.Error("同步 Sidecar 时找不到版本记录", "id", versionID, "error", err)
		return
	}
	res = ver.Resource

	sidecarKey := objectKey + ".meta.json"
	sidecarData := map[string]any{
		"resource_id":   res.ID,
		"resource_name": res.Name,
		"tags":          res.Tags,
		"version_id":    ver.ID,
		"type_key":      res.TypeKey,
		"metadata":      ver.MetaData,
		"synced_at":     time.Now().Format(time.RFC3339),
	}

	if sidecarBytes, err := json.Marshal(sidecarData); err == nil {
		if err := uc.store.Put(ctx, uc.minioConfig, sidecarKey, bytes.NewReader(sidecarBytes), int64(len(sidecarBytes)), "application/json"); err != nil {
			slog.Error("更新 Sidecar 失败", "key", sidecarKey, "error", err)
		} else {
			slog.Debug("Sidecar 刷新成功", "key", sidecarKey)
		}
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

	url, err := uc.store.PresignGet(ctx, uc.minioConfig, v.FilePath, time.Hour)
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

		dv := &ResourceVersionDTO{
			VersionNum: v.VersionNum,
			State:      v.State,
			MetaData:   v.MetaData,
		}
		if v.State == "ACTIVE" {
			url, _ := uc.store.PresignGet(ctx, uc.minioConfig, v.FilePath, time.Hour)
			dv.DownloadURL = url
		}

		cw = append(cw, &ResourceDTO{
			ID:         r.ID,
			TypeKey:    r.TypeKey,
			CategoryID: r.CategoryID,
			Name:       r.Name,
			OwnerID:    r.OwnerID,
			Tags:       r.Tags,
			CreatedAt:  r.CreatedAt,
			LatestVer:  dv,
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

// UpdateResourceTags 更新资源标签 并同步刷新 Sidecar
func (uc *UseCase) UpdateResourceTags(ctx context.Context, id string, tags []string) error {
	return uc.data.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Resource{}).Where("id = ?", id).Select("Tags").Updates(model.Resource{Tags: tags}).Error; err != nil {
			return err
		}

		// 触发异步刷新 Sidecar (获取最新版本)
		var v model.ResourceVersion
		if err := tx.Order("version_num desc").First(&v, "resource_id = ?", id).Error; err == nil {
			uc.dispatchJob(processJob{
				Action:    ActionRefresh,
				ObjectKey: v.FilePath,
				VersionID: v.ID,
			})
		}
		return nil
	})
}

// SyncFromStorage 从存储扫描并同步资源到数据库
func (uc *UseCase) SyncFromStorage(ctx context.Context) (int, error) {
	bucketName := uc.minioConfig
	// 1. 列出所有对象
	// 期望路径格式: resources/{type_key}/{resource_id}/{filename}
	objectCh := uc.store.ListObjects(ctx, bucketName, "resources/", true)

	syncedCount := 0
	for object := range objectCh {
		if strings.HasSuffix(object.Key, ".meta.json") {
			continue // 跳过 Sidecar 文件本身，它们在处理主文件时被读取
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
				slog.Error("无法创建资源主表", "error", err)
				continue
			}
		}

		ver := model.ResourceVersion{
			ResourceID: resourceID,
			VersionNum: 1, // 简单处理，同步默认为 v1
			FileSize:   object.Size,
			FilePath:   object.Key,
			State:      "PENDING",
			MetaData:   map[string]any{"source": "storage_sync"},
		}

		// --- 关键：通过 Sidecar 恢复元数据 ---
		sidecarKey := object.Key + ".meta.json"
		if rc, err := uc.store.Get(ctx, bucketName, sidecarKey); err == nil {
			var sd struct {
				ResourceName string         `json:"resource_name"`
				Tags         []string       `json:"tags"`
				Metadata     map[string]any `json:"metadata"`
			}
			if decodeErr := json.NewDecoder(rc).Decode(&sd); decodeErr == nil {
				res.Name = sd.ResourceName
				res.Tags = sd.Tags
				ver.MetaData = sd.Metadata
			}
			rc.Close()
			// 更新主表（如果已创建）
			uc.data.DB.Save(&res)
		}

		if err := uc.data.DB.Create(&ver).Error; err != nil {
			slog.Error("无法创建版本记录", "error", err)
			continue
		}

		// 5. 触发异步处理器（重新提取元数据和分类）
		uc.dispatchJob(processJob{
			Action:    ActionProcess,
			TypeKey:   typeKey,
			ObjectKey: object.Key,
			VersionID: ver.ID,
		})
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
		if err := uc.store.Delete(ctx, uc.minioConfig, v.FilePath); err != nil {
			slog.Error("无法删除 MinIO 文件", "path", v.FilePath, "error", err)
			continue
		}
		// 删除 Sidecar 元数据文件
		sidecarKey := v.FilePath + ".meta.json"
		if err := uc.store.Delete(ctx, uc.minioConfig, sidecarKey); err != nil {
			slog.Error("无法删除 Sidecar", "path", sidecarKey, "error", err)
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

// ReportProcessResult 由外部 Worker 回调，上报资源处理结果
func (uc *UseCase) ReportProcessResult(ctx context.Context, versionID string, req ProcessResultRequest) error {
	return uc.data.DB.Transaction(func(tx *gorm.DB) error {
		var ver model.ResourceVersion
		if err := tx.First(&ver, "id = ?", versionID).Error; err != nil {
			return err
		}

		// 合并元数据
		if ver.MetaData == nil {
			ver.MetaData = make(map[string]any)
		}
		for k, v := range req.MetaData {
			ver.MetaData[k] = v
		}

		ver.State = req.State
		if err := tx.Save(&ver).Error; err != nil {
			return err
		}

		// 如果处理成功，触发 Sidecar 刷新
		if ver.State == "ACTIVE" {
			uc.dispatchJob(processJob{
				Action:    ActionRefresh,
				ObjectKey: ver.FilePath,
				VersionID: ver.ID,
			})
		}

		slog.Info("接收到处理结果回调", "version_id", versionID, "state", ver.State)
		return nil
	})
}
