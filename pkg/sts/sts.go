package sts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// TokenVendor 处理临时凭证的生成
type TokenVendor struct {
	client    *minio.Client
	accessKey string
	secretKey string
}

func NewTokenVendor(client *minio.Client, ak, sk string) *TokenVendor {
	return &TokenVendor{
		client:    client,
		accessKey: ak,
		secretKey: sk,
	}
}

type STSCredentials struct {
	AccessKey    string    `json:"access_key"`
	SecretKey    string    `json:"secret_key"`
	SessionToken string    `json:"session_token"`
	Expiration   time.Time `json:"expiration"`
}

// GenerateUploadToken 通过 STS AssumeRole 生成临时令牌
func (v *TokenVendor) GenerateUploadToken(ctx context.Context, bucket, prefix string, duration time.Duration) (*STSCredentials, error) {
	// 使用 MinIO SDK 的 STSAssumeRole 提供程序生成凭证
	// 我们使用长期密钥作为“父级”进行操作

	stsOpts := credentials.STSAssumeRoleOptions{
		AccessKey:       v.accessKey,
		SecretKey:       v.secretKey,
		DurationSeconds: int(duration.Seconds()),
	}

	// 创建针对我们自己 MinIO 服务器的提供程序
	if v.client == nil {
		fmt.Println("DEBUG: v.client 为 nil")
		return nil, fmt.Errorf("minio client is nil")
	}
	u := v.client.EndpointURL()
	if u == nil {
		fmt.Println("DEBUG: EndpointURL 返回了 nil")
		return nil, fmt.Errorf("endpoint url is nil")
	}
	endpoint := u.String()
	fmt.Printf("DEBUG: STS 端点: %s\n", endpoint)

	sts, err := credentials.NewSTSAssumeRole(endpoint, stsOpts)
	if err != nil {
		fmt.Printf("DEBUG: NewSTSAssumeRole 错误: %v\n", err)
		return nil, fmt.Errorf("failed to init STS provider: %w", err)
	}

	// 获取凭证
	creds, err := sts.Get()
	if err != nil {
		fmt.Printf("DEBUG: sts.Get 错误: %v\n", err)
		return nil, fmt.Errorf("failed to fetch STS credentials: %w", err)
	}

	return &STSCredentials{
		AccessKey:    creds.AccessKeyID,
		SecretKey:    creds.SecretAccessKey,
		SessionToken: creds.SessionToken,
		Expiration:   time.Now().Add(duration), // 近似值
	}, nil
}

// GeneratePresignedUpload 生成一个 PUT URL
func (v *TokenVendor) GeneratePresignedUpload(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error) {
	url, err := v.client.PresignedPutObject(ctx, bucket, objectName, expiry)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// GenerateDownloadURL 生成一个 GET URL
func (v *TokenVendor) GenerateDownloadURL(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	// reqParams.Set("response-content-disposition", "attachment; filename=\"your-filename.txt\"")
	url, err := v.client.PresignedGetObject(ctx, bucket, objectName, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// FGetObject 下载对象到本地文件
func (v *TokenVendor) FGetObject(ctx context.Context, bucket, objectName, filePath string) error {
	return v.client.FGetObject(ctx, bucket, objectName, filePath, minio.GetObjectOptions{})
}

// ListObjects 列出前缀下的所有对象
func (v *TokenVendor) ListObjects(ctx context.Context, bucket, prefix string) <-chan minio.ObjectInfo {
	return v.client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix: prefix,
	})
}

// StatObject 获取对象元数据 (检查是否存在)
func (v *TokenVendor) StatObject(ctx context.Context, bucket, objectName string) (minio.ObjectInfo, error) {
	return v.client.StatObject(ctx, bucket, objectName, minio.StatObjectOptions{})
}

// RemoveObject 删除存储对象
func (v *TokenVendor) RemoveObject(ctx context.Context, bucket, objectName string) error {
	return v.client.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
}

// PutObjectJSON 将对象作为 JSON 上传到存储
func (v *TokenVendor) PutObjectJSON(ctx context.Context, bucket, objectName string, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = v.client.PutObject(ctx, bucket, objectName, bytes.NewReader(b), int64(len(b)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	return err
}
