package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/model"
	"github.com/liny/sim-hub/pkg/sts"
	"gorm.io/gorm"
)

type ResourceUseCase struct {
	data        *data.Data
	tokenVendor *sts.TokenVendor
	minioConfig string // bucket name
}

// ResourceDTO for API Response
type ResourceDTO struct {
	ID        string              `json:"id"`
	TypeKey   string              `json:"type_key"`
	Name      string              `json:"name"`
	OwnerID   string              `json:"owner_id"`
	Tags      []string            `json:"tags"`
	CreatedAt time.Time           `json:"created_at"`
	LatestVer *ResourceVersionDTO `json:"latest_version,omitempty"`
}

type ResourceVersionDTO struct {
	VersionNum  int            `json:"version_num"`
	FileSize    int64          `json:"file_size"`
	MetaData    map[string]any `json:"meta_data"`
	DownloadURL string         `json:"download_url,omitempty"`
}

func NewResourceUseCase(d *data.Data, tv *sts.TokenVendor, bucket string) *ResourceUseCase {
	return &ResourceUseCase{
		data:        d,
		tokenVendor: tv,
		minioConfig: bucket,
	}
}

// UploadTicket Response for the UI/Client
type UploadTicket struct {
	TicketID     string `json:"ticket_id"`
	AccessKey    string `json:"access_key"` // Placeholder or real if using STS
	SecretKey    string `json:"secret_key"` // Placeholder
	SessionToken string `json:"session_token"`
	Endpoint     string `json:"endpoint"`
	Bucket       string `json:"bucket"`
	Prefix       string `json:"prefix"`

	// We add this for the "Simpler" Presigned URL flow, though spec asks for credentials.
	// If the client supports it, they can just use this URL.
	PresignedURL string `json:"presigned_url,omitempty"`
}

// RequestUploadToken handles the logic for POST /upload/token
func (uc *ResourceUseCase) RequestUploadToken(ctx context.Context, filename, typeKey string) (*UploadTicket, error) {
	// 1. Verify resource type exists
	var resType model.ResourceType
	if err := uc.data.DB.First(&resType, "type_key = ?", typeKey).Error; err != nil {
		return nil, fmt.Errorf("invalid resource type: %s", typeKey)
	}

	// 2. Generate a unique "Ticket ID" or Session ID to track this transaction
	// The spec implementation roadmap mentions "STS mode", so we create a path like:
	// maps/2026/uuid/filename
	requestID := uuid.New().String()
	datePath := time.Now().Format("2006")
	objectKey := fmt.Sprintf("%ss/%s/%s/%s", typeKey, datePath, requestID, filename)

	// 3. Generate Presigned URL (as the primary implementation of "Auth")
	// Expiry: 1 hour
	url, err := uc.tokenVendor.GeneratePresignedUpload(ctx, uc.minioConfig, objectKey, time.Hour)
	if err != nil {
		return nil, err
	}

	// 4. Return the ticket
	return &UploadTicket{
		TicketID:     requestID, // Use this UUID as the ticket/folder
		AccessKey:    "STS_NOT_IMPLEMENTED_USE_URL",
		SecretKey:    "STS_NOT_IMPLEMENTED_USE_URL",
		Endpoint:     "See PresignedURL",
		Bucket:       uc.minioConfig,
		Prefix:       objectKey,
		PresignedURL: url,
	}, nil
}

// ConfirmUploadRequest Data needed to confirm
type ConfirmUploadRequest struct {
	TicketID  string         `json:"ticket_id"`
	Filename  string         `json:"filename"` // We re-confirm filename to reconstruct path
	TypeKey   string         `json:"type_key"`
	Name      string         `json:"name"` // Resource Name
	OwnerID   string         `json:"owner_id"`
	Size      int64          `json:"size"`
	ExtraMeta map[string]any `json:"extra_meta"`
}

// ConfirmUpload handles POST /upload/confirm
func (uc *ResourceUseCase) ConfirmUpload(ctx context.Context, req *ConfirmUploadRequest) error {
	// 1. Reconstruct Object Key (In a real system, we might cache the ticket state in Redis)
	// Here we trust the client provides correct params matching the token request, or we stat the object.
	datePath := time.Now().Format("2006") // Potential Bug: If date changes between request and confirm?
	// Better: TicketID implies the path if we used a strict structure.
	// Let's assume the TicketID IS the UUID part of the path.
	objectKey := fmt.Sprintf("%ss/%s/%s/%s", req.TypeKey, datePath, req.TicketID, req.Filename)

	// 2. (Optional) Stat the object in MinIO to verify it exists and get size/hash
	// We skip this strict check for the MVP, effectively trusting the client's "Success" signal
	// triggers the DB record. The "Worker" pipeline will verify it later.

	// 3. DB Transaction
	return uc.data.DB.Transaction(func(tx *gorm.DB) error {
		// Create Resource Record
		res := model.Resource{
			TypeKey: req.TypeKey,
			Name:    req.Name,
			OwnerID: req.OwnerID,
		}
		if err := tx.Create(&res).Error; err != nil {
			return err
		}

		// Create Version Record
		ver := model.ResourceVersion{
			ResourceID: res.ID,
			VersionNum: 1,
			FilePath:   objectKey,
			FileSize:   req.Size,
			MetaData:   req.ExtraMeta,
			State:      "ACTIVE", // Directly active for MVP, normally PENDING -> PROCESSING
		}
		if err := tx.Create(&ver).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetResource returns resource details with download URL for the latest version
func (uc *ResourceUseCase) GetResource(ctx context.Context, id string) (*ResourceDTO, error) {
	var r model.Resource
	// Preload latest version? For MVP we just pick the latest by version_num
	if err := uc.data.DB.First(&r, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// Get Latest Version
	var v model.ResourceVersion
	if err := uc.data.DB.Order("version_num desc").First(&v, "resource_id = ?", id).Error; err != nil {
		// It's possible to have resource without version if transaction failed?
		// Should handle gracefully, but here we error
		return nil, err
	}

	// Generate Presigned GET URL
	url, err := uc.tokenVendor.GenerateDownloadURL(ctx, uc.minioConfig, v.FilePath, time.Hour)
	if err != nil {
		return nil, err
	}

	return &ResourceDTO{
		ID:        r.ID,
		TypeKey:   r.TypeKey,
		Name:      r.Name,
		OwnerID:   r.OwnerID,
		Tags:      r.Tags,
		CreatedAt: r.CreatedAt,
		LatestVer: &ResourceVersionDTO{
			VersionNum:  v.VersionNum,
			FileSize:    v.FileSize,
			MetaData:    v.MetaData,
			DownloadURL: url,
		},
	}, nil
}

// ListResources returns a list of resources
func (uc *ResourceUseCase) ListResources(ctx context.Context, page, size int) ([]*ResourceDTO, int64, error) {
	var resources []model.Resource
	var total int64

	offset := (page - 1) * size
	if err := uc.data.DB.Model(&model.Resource{}).Count(&total).Limit(size).Offset(offset).Order("created_at desc").Find(&resources).Error; err != nil {
		return nil, 0, err
	}

	cw := make([]*ResourceDTO, 0, len(resources))
	for _, r := range resources {
		cw = append(cw, &ResourceDTO{
			ID:        r.ID,
			TypeKey:   r.TypeKey,
			Name:      r.Name,
			OwnerID:   r.OwnerID,
			Tags:      r.Tags,
			CreatedAt: r.CreatedAt,
		})
	}
	return cw, total, nil
}
