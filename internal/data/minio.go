package data

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/liny/sim-hub/internal/conf"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOClient struct {
	Client *minio.Client
	Config conf.MinIO
}

func NewMinIO(c *conf.MinIO) (*MinIOClient, error) {
	minioClient, err := minio.New(c.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKey, c.SecretKey, ""),
		Secure: c.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	// 连通性测试
	if _, err := minioClient.ListBuckets(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to minio: %w", err)
	}

	// 确保存储桶 (Bucket) 已存在
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, c.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		slog.Info("存储桶不存在，正在尝试创建", "bucket", c.Bucket)
		err = minioClient.MakeBucket(ctx, c.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		slog.Info("存储桶创建成功", "bucket", c.Bucket)

		// 说明：MinIO 桶策略默认为 Private，仅通过 STS 或预签名 URL 访问，
		// 因此此处无需显式设置公共策略。
	}

	return &MinIOClient{
		Client: minioClient,
		Config: *c,
	}, nil
}
