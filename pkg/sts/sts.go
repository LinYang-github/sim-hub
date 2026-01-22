package sts

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// TokenVendor handles generation of temporary credentials
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

// GenerateUploadToken generates a temporary token via STS AssumeRole
func (v *TokenVendor) GenerateUploadToken(ctx context.Context, bucket, prefix string, duration time.Duration) (*STSCredentials, error) {
	// Use MinIO SDK's STSAssumeRole provider to generate credentials
	// We act as the "Parent" using our long-term keys

	stsOpts := credentials.STSAssumeRoleOptions{
		AccessKey:       v.accessKey,
		SecretKey:       v.secretKey,
		DurationSeconds: int(duration.Seconds()),
	}

	// Create a provider targeting our own MinIO server
	if v.client == nil {
		fmt.Println("DEBUG: v.client is nil")
		return nil, fmt.Errorf("minio client is nil")
	}
	u := v.client.EndpointURL()
	if u == nil {
		fmt.Println("DEBUG: EndpointURL returned nil")
		return nil, fmt.Errorf("endpoint url is nil")
	}
	endpoint := u.String()
	fmt.Printf("DEBUG: STS Endpoint: %s\n", endpoint)

	sts, err := credentials.NewSTSAssumeRole(endpoint, stsOpts)
	if err != nil {
		fmt.Printf("DEBUG: NewSTSAssumeRole error: %v\n", err)
		return nil, fmt.Errorf("failed to init STS provider: %w", err)
	}

	// Fetch the credentials
	creds, err := sts.Get()
	if err != nil {
		fmt.Printf("DEBUG: sts.Get error: %v\n", err)
		return nil, fmt.Errorf("failed to fetch STS credentials: %w", err)
	}

	return &STSCredentials{
		AccessKey:    creds.AccessKeyID,
		SecretKey:    creds.SecretAccessKey,
		SessionToken: creds.SessionToken,
		Expiration:   time.Now().Add(duration), // Approximation
	}, nil
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
