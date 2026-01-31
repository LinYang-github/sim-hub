package core

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"sim-hub/internal/data"
	"sim-hub/pkg/storage"
	"gorm.io/gorm"
)

type UploadManager struct {
	data        *data.Data // 需要 DB 事务
	store       storage.MultipartBlobStore
	stsProvider storage.SecurityTokenProvider
	bucket      string
	registrar   *ResourceWriter // 使用具体类型简化，或者用 ResourceRegistrar 接口
}

func NewUploadManager(d *data.Data, store storage.MultipartBlobStore, stsProvider storage.SecurityTokenProvider, bucket string, registrar *ResourceWriter) *UploadManager {
	return &UploadManager{
		data:        d,
		store:       store,
		stsProvider: stsProvider,
		bucket:      bucket,
		registrar:   registrar,
	}
}

// RequestUploadToken 请求上传令牌
func (u *UploadManager) RequestUploadToken(ctx context.Context, req ApplyUploadTokenRequest) (*UploadTicket, error) {
	if u.store == nil {
		return nil, fmt.Errorf("storage service (MinIO) is not available, please check server status")
	}
	ticketID := uuid.New().String()
	// objectKey 格式: resources/{type}/{uuid}/{filename}
	// objectKey 格式: resources/{type}/{uuid}/{filename}
	objectKey := "resources/" + req.ResourceType + "/" + ticketID + "/" + req.Filename

	if req.Mode == "sts" {
		if u.stsProvider == nil {
			return nil, fmt.Errorf("sts provider not configured")
		}
		creds, err := u.stsProvider.GenerateSTSToken(ctx, u.bucket, objectKey, time.Hour)
		if err != nil {
			return nil, err
		}
		return &UploadTicket{
			TicketID:    ticketID + "::" + objectKey,
			Credentials: creds,
			Bucket:      u.bucket,
			ObjectKey:   objectKey,
		}, nil
	}

	// 默认模式: 预签名 URL
	url, err := u.store.PresignPut(ctx, u.bucket, objectKey, time.Hour)
	if err != nil {
		return nil, err
	}

	return &UploadTicket{
		TicketID:     ticketID + "::" + objectKey,
		PresignedURL: url,
	}, nil
}

// InitMultipartUpload 初始化分片上传
func (u *UploadManager) InitMultipartUpload(ctx context.Context, req InitMultipartUploadRequest) (*InitMultipartUploadResponse, error) {
	if u.store == nil {
		return nil, fmt.Errorf("storage service (MinIO) is not available")
	}
	ticketID := uuid.New().String()
	objectKey := "resources/" + req.ResourceType + "/" + ticketID + "/" + req.Filename

	uploadID, err := u.store.InitMultipart(ctx, u.bucket, objectKey)
	if err != nil {
		slog.ErrorContext(ctx, "初始化分片上传失败", "error", err, "key", objectKey)
		return nil, err
	}

	return &InitMultipartUploadResponse{
		TicketID:  ticketID + "::" + objectKey,
		UploadID:  uploadID,
		Bucket:    u.bucket,
		ObjectKey: objectKey,
	}, nil
}

// GetMultipartUploadPartURL 获取分片上传的预签名 URL
func (u *UploadManager) GetMultipartUploadPartURL(ctx context.Context, req GetPartURLRequest) (*GetPartURLResponse, error) {
	if u.store == nil {
		return nil, fmt.Errorf("storage service (MinIO) is not available")
	}
	objectKey := ""
	if len(req.TicketID) > 38 {
		objectKey = req.TicketID[38:]
	}

	url, err := u.store.PresignPart(ctx, u.bucket, objectKey, req.UploadID, req.PartNumber, time.Hour)
	if err != nil {
		slog.ErrorContext(ctx, "生成分片上传 URL 失败", "error", err, "key", objectKey, "part", req.PartNumber)
		return nil, err
	}

	return &GetPartURLResponse{URL: url}, nil
}

// CompleteMultipartUpload 完成分片上传并注册资源
func (u *UploadManager) CompleteMultipartUpload(ctx context.Context, req CompleteMultipartUploadRequest) error {
	if u.store == nil {
		return fmt.Errorf("storage service (MinIO) is not available")
	}
	objectKey := ""
	if len(req.TicketID) > 38 {
		objectKey = req.TicketID[38:]
	}

	// 1. 在存储层完成分片合并
	if err := u.store.CompleteMultipart(ctx, u.bucket, objectKey, req.UploadID, req.Parts); err != nil {
		slog.ErrorContext(ctx, "完成分片上传失败", "error", err, "key", objectKey, "upload_id", req.UploadID)
		return err
	}

	// 2. 获取最终对象信息（获取真实大小）
	objInfo, err := u.store.Stat(ctx, u.bucket, objectKey)
	if err != nil {
		slog.ErrorContext(ctx, "无法获取合并后对象信息", "key", objectKey, "error", err)
		return fmt.Errorf("uploaded file not found after completion: %w", err)
	}

	// 3. 注册到数据库
	return u.data.DB.Transaction(func(tx *gorm.DB) error {
		return u.registrar.CreateResourceAndVersion(tx, req.TypeKey, req.CategoryID, req.Name, req.OwnerID, req.Scope, objectKey, objInfo.Size, req.Tags, req.SemVer, req.Dependencies, req.ExtraMeta)
	})
}

// ConfirmUpload 确认上传完成
func (u *UploadManager) ConfirmUpload(ctx context.Context, req ConfirmUploadRequest) error {
	if u.store == nil {
		return fmt.Errorf("storage service (MinIO) is not available")
	}
	objectKey := ""
	if len(req.TicketID) > 38 {
		objectKey = req.TicketID[38:]
	}

	// 0. 验证 MinIO 中对象是否存在
	slog.InfoContext(ctx, "Checking object existence", "bucket", u.bucket, "key", objectKey)
	objInfo, err := u.store.Stat(ctx, u.bucket, objectKey)
	if err != nil {
		slog.ErrorContext(ctx, "无法获取对象信息", "key", objectKey, "error", err)
		return fmt.Errorf("uploaded file not found: %w", err)
	}
	slog.InfoContext(ctx, "Object found", "size", objInfo.Size)

	return u.data.DB.Transaction(func(tx *gorm.DB) error {
		slog.InfoContext(ctx, "Starting DB transaction for resource creation", "name", req.Name)
		err := u.registrar.CreateResourceAndVersion(tx, req.TypeKey, req.CategoryID, req.Name, req.OwnerID, req.Scope, objectKey, objInfo.Size, req.Tags, req.SemVer, req.Dependencies, req.ExtraMeta)
		if err != nil {
			slog.ErrorContext(ctx, "CreateResourceAndVersion failed", "error", err)
			return err
		}
		return nil
	})
}
