package core

import (
	"time"

	"github.com/liny/sim-hub/pkg/storage"
)

// DTOs 数据传输对象

type ApplyUploadTokenRequest struct {
	ResourceType string `json:"resource_type"`
	Checksum     string `json:"checksum"`
	Size         int64  `json:"size"`
	Filename     string `json:"filename"`
	Mode         string `json:"mode"` // "presigned" (默认) 或 "sts"
}

type ConfirmUploadRequest struct {
	TicketID     string          `json:"ticket_id"`
	TypeKey      string          `json:"type_key"`
	CategoryID   string          `json:"category_id"`
	Name         string          `json:"name"`
	OwnerID      string          `json:"owner_id"`
	Scope        string          `json:"scope"`
	Tags         []string        `json:"tags"`
	Size         int64           `json:"size"`
	SemVer       string          `json:"semver"`       // 新增：版本号
	Dependencies []DependencyDTO `json:"dependencies"` // 新增：依赖列表
	ExtraMeta    map[string]any  `json:"extra_meta"`
}

type DependencyDTO struct {
	TargetResourceID string `json:"target_resource_id"`
	Constraint       string `json:"constraint"`
}

type UpdateResourceTagsRequest struct {
	Tags []string `json:"tags"`
}

type UpdateResourceScopeRequest struct {
	Scope string `json:"scope"`
}

type UpdateResourceRequest struct {
	Name       string `json:"name,omitempty"`
	CategoryID string `json:"category_id,omitempty"`
}

type UpdateVersionMetadataRequest struct {
	MetaData map[string]any `json:"meta_data"`
}

// Multipart Upload DTOs
type InitMultipartUploadRequest struct {
	ResourceType string `json:"resource_type"`
	Filename     string `json:"filename"`
}

type InitMultipartUploadResponse struct {
	TicketID  string `json:"ticket_id"`
	UploadID  string `json:"upload_id"`
	Bucket    string `json:"bucket"`
	ObjectKey string `json:"object_key"`
}

type GetPartURLRequest struct {
	TicketID   string `json:"ticket_id"`
	UploadID   string `json:"upload_id"`
	PartNumber int    `json:"part_number"`
}

type GetPartURLResponse struct {
	URL string `json:"url"`
}

type ProcessResultRequest struct {
	MetaData map[string]any `json:"meta_data"`
	State    string         `json:"state"` // ACTIVE, ERROR
	Message  string         `json:"message,omitempty"`
}

type CompleteMultipartUploadRequest struct {
	TicketID     string          `json:"ticket_id"`
	UploadID     string          `json:"upload_id"`
	Parts        []storage.Part  `json:"parts"`
	TypeKey      string          `json:"type_key"`
	CategoryID   string          `json:"category_id"`
	Name         string          `json:"name"`
	OwnerID      string          `json:"owner_id"`
	Scope        string          `json:"scope"`
	Tags         []string        `json:"tags"`
	SemVer       string          `json:"semver"`       // 新增：版本号
	Dependencies []DependencyDTO `json:"dependencies"` // 新增：依赖列表
	ExtraMeta    map[string]any  `json:"extra_meta"`
}

type CategoryDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
}

type CreateCategoryRequest struct {
	TypeKey  string `json:"type_key"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
}

type UpdateCategoryRequest struct {
	Name     string  `json:"name,omitempty"`
	ParentID *string `json:"parent_id,omitempty"`
}

type UploadTicket struct {
	TicketID     string                  `json:"ticket_id"`
	PresignedURL string                  `json:"presigned_url"`
	Credentials  *storage.STSCredentials `json:"credentials,omitempty"`
	Bucket       string                  `json:"bucket,omitempty"`
	ObjectKey    string                  `json:"object_key,omitempty"`
}

type ResourceDTO struct {
	ID         string              `json:"id"`
	TypeKey    string              `json:"type_key"`
	CategoryID string              `json:"category_id,omitempty"`
	Name       string              `json:"name"`
	OwnerID    string              `json:"owner_id"`
	Scope      string              `json:"scope"` // 新增：作用域
	Tags       []string            `json:"tags"`
	CreatedAt  time.Time           `json:"created_at"`
	LatestVer  *ResourceVersionDTO `json:"latest_version,omitempty"`
}

type ResourceVersionDTO struct {
	ID          string         `json:"id"`
	VersionNum  int            `json:"version_num"`
	SemVer      string         `json:"semver"`
	FileSize    int64          `json:"file_size"`
	MetaData    map[string]any `json:"meta_data"`
	State       string         `json:"state"`
	DownloadURL string         `json:"download_url,omitempty"`
}
