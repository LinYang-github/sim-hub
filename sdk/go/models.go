package simhub

import "time"

type ResourceType struct {
	TypeKey         string         `json:"type_key"`
	TypeName        string         `json:"type_name"`
	SchemaDef       map[string]any `json:"schema_def"`
	CategoryMode    string         `json:"category_mode"`
	IntegrationMode string         `json:"integration_mode"`
	UploadMode      string         `json:"upload_mode"`
	CreatedAt       time.Time      `json:"created_at"`
}

type Category struct {
	ID       string `json:"id"`
	TypeKey  string `json:"type_key"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
}

type ResourceVersion struct {
	ID          string         `json:"id"`
	VersionNum  int            `json:"version_num"`
	SemVer      string         `json:"semver"`
	FileSize    int64          `json:"file_size"`
	MetaData    map[string]any `json:"meta_data"`
	State       string         `json:"state"`
	DownloadURL string         `json:"download_url"`
}

type Resource struct {
	ID              string          `json:"id"`
	TypeKey         string          `json:"type_key"`
	Name            string          `json:"name"`
	OwnerID         string          `json:"owner_id"`
	Scope           string          `json:"scope"`
	Tags            []string        `json:"tags"`
	LatestVersionID string          `json:"latest_version_id"`
	LatestVersion   ResourceVersion `json:"latest_version"`
	CreatedAt       time.Time       `json:"created_at"`
}

type ResourceListResponse struct {
	Items []Resource `json:"items"`
	Total int64      `json:"total"`
}

type UploadTokenResponse struct {
	TicketID     string `json:"ticket_id"`
	PresignedURL string `json:"presigned_url"`
}

type MultipartInitResponse struct {
	UploadID string `json:"upload_id"`
	Key      string `json:"key"`
}

type PartETag struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}
