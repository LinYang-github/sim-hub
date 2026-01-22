package sts

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

// TokenVendor handles generation of temporary credentials
type TokenVendor struct {
	client *minio.Client
}

func NewTokenVendor(client *minio.Client) *TokenVendor {
	return &TokenVendor{client: client}
}

type STSCredentials struct {
	AccessKey    string    `json:"access_key"`
	SecretKey    string    `json:"secret_key"`
	SessionToken string    `json:"session_token"`
	Expiration   time.Time `json:"expiration"`
}

// GenerateUploadToken generates a temporary token for uploading a specific object
func (v *TokenVendor) GenerateUploadToken(ctx context.Context, bucket, prefix string, duration time.Duration) (*STSCredentials, error) {
	// Policy definition removed as it was unused in Presigned URL implementation
	// Using MinIO Admin or core client... (omitted comments)
	return nil, fmt.Errorf("STS implementation pending clearer IAM setup; Use PresignedURL for specific object upload")
}

// GeneratePresignedUpload generates a PUT URL
func (v *TokenVendor) GeneratePresignedUpload(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error) {
	url, err := v.client.PresignedPutObject(ctx, bucket, objectName, expiry)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// GenerateDownloadURL generates a GET URL
func (v *TokenVendor) GenerateDownloadURL(ctx context.Context, bucket, objectName string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	// reqParams.Set("response-content-disposition", "attachment; filename=\"your-filename.txt\"")
	url, err := v.client.PresignedGetObject(ctx, bucket, objectName, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}
