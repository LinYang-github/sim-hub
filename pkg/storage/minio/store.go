package minio

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/liny/sim-hub/pkg/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStore 实现了 storage.MultipartBlobStore 接口
type MinIOStore struct {
	client    *minio.Client
	accessKey string
	secretKey string
}

func NewMinIOStore(client *minio.Client, ak, sk string) *MinIOStore {
	return &MinIOStore{
		client:    client,
		accessKey: ak,
		secretKey: sk,
	}
}

// --- Basic Operations ---

func (s *MinIOStore) Stat(ctx context.Context, bucket, key string) (*storage.ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}
	return &storage.ObjectInfo{
		Key:          info.Key,
		Size:         info.Size,
		ETag:         info.ETag,
		LastModified: info.LastModified,
		ContentType:  info.ContentType,
		Metadata:     info.UserMetadata,
	}, nil
}

func (s *MinIOStore) Delete(ctx context.Context, bucket, key string) error {
	return s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

func (s *MinIOStore) ListObjects(ctx context.Context, bucket, prefix string, recursive bool) <-chan storage.ObjectInfo {
	outCh := make(chan storage.ObjectInfo)
	go func() {
		defer close(outCh)
		opts := minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: recursive,
		}
		for info := range s.client.ListObjects(ctx, bucket, opts) {
			if info.Err != nil {
				// 简单的错误处理：跳过或记录日志?
				// 由于接口只返回 channel，我们这里无法直接返回 error。
				// 通常的做法是 item 结构体包含 error 字段，或者记录日志。
				// 这里暂时由消费者处理 zero value 或添加 error 字段到 ObjectInfo。
				continue
			}
			outCh <- storage.ObjectInfo{
				Key:          info.Key,
				Size:         info.Size,
				ETag:         info.ETag,
				LastModified: info.LastModified,
				ContentType:  info.ContentType,
				Metadata:     info.UserMetadata,
			}
		}
	}()
	return outCh
}

func (s *MinIOStore) Put(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *MinIOStore) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
}

func (s *MinIOStore) DownloadFile(ctx context.Context, bucket, key, localPath string) error {
	// FGetObject 内部会处理并发下载和校验
	return s.client.FGetObject(ctx, bucket, key, localPath, minio.GetObjectOptions{})
}

func (s *MinIOStore) PresignPut(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedPutObject(ctx, bucket, key, expiry)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *MinIOStore) PresignGet(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	u, err := s.client.PresignedGetObject(ctx, bucket, key, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// --- Multipart Operations ---

func (s *MinIOStore) InitMultipart(ctx context.Context, bucket, key string) (string, error) {
	// MinIO Core API 提供了更底层的分片上传控制
	// 但 minio-go v7 封装较深，标准操作是 PutObject 自动分片。
	// 若要显式控制分片上传 (Init/Part/Complete)，需要使用 Core 客户端。
	core := minio.Core{Client: s.client}
	return core.NewMultipartUpload(ctx, bucket, key, minio.PutObjectOptions{})
}

func (s *MinIOStore) PresignPart(ctx context.Context, bucket, key, uploadID string, partNumber int, expiry time.Duration) (string, error) {
	// 对于 MinIO/S3，Part 上传的预签名 URL 本质上也是 PutObject URL，但带有 patchNumber 和 uploadId 参数
	reqParams := make(url.Values)
	reqParams.Set("uploadId", uploadID)
	reqParams.Set("partNumber", fmt.Sprintf("%d", partNumber))

	// 使用 PresignedPutObject，但通过特殊的 trick (UserMetadata?) 或者手动构造？
	// MinIO SDK 的 PresignedPutObject 不直接支持 passing query params for uploadId.
	// 我们需要使用 client.Presign (Generic) 或者构建自定义请求。

	// 更稳妥的方式是使用 Presign 方法
	u, err := s.client.Presign(ctx, "PUT", bucket, key, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *MinIOStore) CompleteMultipart(ctx context.Context, bucket, key, uploadID string, parts []storage.Part) error {
	core := minio.Core{Client: s.client}

	// Convert storage.Part to minio.CompletePart
	minioParts := make([]minio.CompletePart, len(parts))
	for i, p := range parts {
		minioParts[i] = minio.CompletePart{
			PartNumber: p.PartNumber,
			ETag:       p.ETag,
		}
	}

	_, err := core.CompleteMultipartUpload(ctx, bucket, key, uploadID, minioParts, minio.PutObjectOptions{})
	return err
}

func (s *MinIOStore) AbortMultipart(ctx context.Context, bucket, key, uploadID string) error {
	core := minio.Core{Client: s.client}
	return core.AbortMultipartUpload(ctx, bucket, key, uploadID)
}

// --- Security Token Operations ---

func (s *MinIOStore) GenerateSTSToken(ctx context.Context, bucket, prefix string, duration time.Duration) (*storage.STSCredentials, error) {
	stsOpts := credentials.STSAssumeRoleOptions{
		AccessKey:       s.accessKey,
		SecretKey:       s.secretKey,
		DurationSeconds: int(duration.Seconds()),
	}

	u := s.client.EndpointURL()
	if u == nil {
		return nil, fmt.Errorf("minio endpoint url is nil")
	}

	stsProvider, err := credentials.NewSTSAssumeRole(u.String(), stsOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to init STS provider: %w", err)
	}

	// 获取凭证
	creds, err := stsProvider.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch STS credentials: %w", err)
	}

	return &storage.STSCredentials{
		AccessKey:    creds.AccessKeyID,
		SecretKey:    creds.SecretAccessKey,
		SessionToken: creds.SessionToken,
		Expiration:   time.Now().Add(duration), // 近似值
	}, nil
}
