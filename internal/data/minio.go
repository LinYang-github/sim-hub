package data

import (
	"context"
	"fmt"
	"log"

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

	// Check connection
	if _, err := minioClient.ListBuckets(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to minio: %w", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, c.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		log.Printf("Bucket %s does not exist, attempting to create...", c.Bucket)
		err = minioClient.MakeBucket(ctx, c.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to create bucket: %w", err)
		}
		log.Printf("Bucket %s created successfully", c.Bucket)

		// Set policy to download (public read for processed/preview? or keep all private?)
		// Spec says: MinIO 桶策略设置为 Private，仅通过 STS 或预签名 URL 访问。
		// So no need to set public policy here.
	}

	return &MinIOClient{
		Client: minioClient,
		Config: *c,
	}, nil
}
