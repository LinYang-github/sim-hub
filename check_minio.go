package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	endpoint := "localhost:9000"
	accessKeyID := "minioadmin"
	secretAccessKey := "minioadmin"
	useSSL := false

	// 初使化 minio client 对象。
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		slog.Error("MinIO 初始化失败", "error", err)
		return
	}

	bucketName := "simhub-raw"
	ctx := context.Background()

	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil || !exists {
		fmt.Printf("Bucket %s does not exist\n", bucketName)
		return
	}

	objectCh := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Recursive: true,
	})

	count := 0
	for object := range objectCh {
		if object.Err != nil {
			slog.Error("ListObjects 错误", "error", object.Err)
			return
		}
		fmt.Println("-", object.Key)
		count++
	}
	fmt.Printf("\nTotal files in MinIO: %d\n", count)
}
