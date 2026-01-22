package storage

import (
	"context"
	"io"
	"time"
)

// ObjectInfo 通用对象元数据
type ObjectInfo struct {
	Key          string
	Size         int64
	ETag         string
	LastModified time.Time
	ContentType  string
	Metadata     map[string]string
}

// Part 分片信息
type Part struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

// BlobStore 核心存储接口
type BlobStore interface {
	// 基础操作
	Stat(ctx context.Context, bucket, key string) (*ObjectInfo, error)
	Delete(ctx context.Context, bucket, key string) error

	// 列表操作
	ListObjects(ctx context.Context, bucket, prefix string, recursive bool) <-chan ObjectInfo

	// 流式 IO (尽量避免直接读取 bytes，使用 Reader/Writer)
	Put(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error
	Get(ctx context.Context, bucket, key string) (io.ReadCloser, error)

	// 本地文件优化操作 (Zero-copy or optimized transfer)
	DownloadFile(ctx context.Context, bucket, key, localPath string) error

	// 预签名 (URL Vending)
	PresignPut(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
	PresignGet(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
}

// MultipartBlobStore 分片上传扩展接口
// 用于支持超大文件或断点续传
type MultipartBlobStore interface {
	BlobStore

	// 初始化分片上传，返回 UploadID
	InitMultipart(ctx context.Context, bucket, key string) (string, error)

	// 生成分片上传的预签名 URL
	PresignPart(ctx context.Context, bucket, key, uploadID string, partNumber int, expiry time.Duration) (string, error)

	// 完成分片上传
	CompleteMultipart(ctx context.Context, bucket, key, uploadID string, parts []Part) error

	// 取消分片上传
	AbortMultipart(ctx context.Context, bucket, key, uploadID string) error
}

// STSCredentials 临时安全凭证
type STSCredentials struct {
	AccessKey    string    `json:"access_key"`
	SecretKey    string    `json:"secret_key"`
	SessionToken string    `json:"session_token"`
	Expiration   time.Time `json:"expiration"`
}

// SecurityTokenProvider 用于生成直传临时凭证 (STS)
// 这是一个特定于云厂商(AWS/Aliyun/MinIO)的能力接口
type SecurityTokenProvider interface {
	GenerateSTSToken(ctx context.Context, bucket, prefix string, duration time.Duration) (*STSCredentials, error)
}
