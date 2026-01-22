package resource

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/liny/sim-hub/internal/core/module"
	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/model"
	"github.com/liny/sim-hub/pkg/sts"
	"gorm.io/gorm"
)

// Module implements module.Module
type Module struct {
	uc *UseCase
}

func NewModule(d *data.Data, tv *sts.TokenVendor, bucket string) module.Module {
	return &Module{
		uc: NewUseCase(d, tv, bucket),
	}
}

func (m *Module) RegisterRoutes(g *gin.RouterGroup) {
	// /api/v1/integration/upload/...
	integration := g.Group("/integration")
	{
		integration.POST("/upload/token", m.ApplyUploadToken)
		integration.POST("/upload/confirm", m.ConfirmUpload)
	}

	// /api/v1/resources
	resources := g.Group("/resources")
	{
		resources.GET("", m.ListResources)
		resources.GET("/:id", m.GetResource)
	}
}

// --- Service Handlers (Moved from internal/service/upload.go) ---

func (m *Module) ApplyUploadToken(c *gin.Context) {
	var req ApplyUploadTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ticket, err := m.uc.RequestUploadToken(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ticket)
}

func (m *Module) ConfirmUpload(c *gin.Context) {
	var req ConfirmUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := m.uc.ConfirmUpload(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Processing started"})
}

func (m *Module) GetResource(c *gin.Context) {
	id := c.Param("id")
	res, err := m.uc.GetResource(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (m *Module) ListResources(c *gin.Context) {
	page := 1
	size := 20
	list, total, err := m.uc.ListResources(c.Request.Context(), page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items": list,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// --- UseCase Logic (Moved from internal/biz/resource.go) ---

type UseCase struct {
	data        *data.Data
	tokenVendor *sts.TokenVendor
	minioConfig string
}

func NewUseCase(d *data.Data, tv *sts.TokenVendor, bucket string) *UseCase {
	return &UseCase{data: d, tokenVendor: tv, minioConfig: bucket}
}

// DTOs
type ApplyUploadTokenRequest struct {
	ResourceType string `json:"resource_type"`
	Checksum     string `json:"checksum"`
	Size         int64  `json:"size"`
	Filename     string `json:"filename"`
}

type ConfirmUploadRequest struct {
	TicketID  string         `json:"ticket_id"`
	TypeKey   string         `json:"type_key"`
	Name      string         `json:"name"`
	OwnerID   string         `json:"owner_id"`
	Size      int64          `json:"size"`
	ExtraMeta map[string]any `json:"extra_meta"`
}

type UploadTicket struct {
	TicketID     string              `json:"ticket_id"`
	PresignedURL string              `json:"presigned_url"`
	Credentials  *sts.STSCredentials `json:"credentials,omitempty"`
}

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

// Logic Methods
func (uc *UseCase) RequestUploadToken(ctx context.Context, req ApplyUploadTokenRequest) (*UploadTicket, error) {
	ticketID := uuid.New().String()
	// objectKey: resources/{type}/{uuid}/{filename}
	objectKey := "resources/" + req.ResourceType + "/" + ticketID + "/" + req.Filename

	url, err := uc.tokenVendor.GeneratePresignedUpload(ctx, uc.minioConfig, objectKey, time.Hour)
	if err != nil {
		return nil, err
	}

	return &UploadTicket{
		TicketID:     ticketID + "::" + objectKey, // Simple storage for stateless verify (in prod use Redis)
		PresignedURL: url,
	}, nil
}

func (uc *UseCase) ConfirmUpload(ctx context.Context, req ConfirmUploadRequest) error {
	// Parse ticket (simple split for MVP)
	// In real world, verify Redis or JWT
	// ticketParts := strings.Split(req.TicketID, "::")
	objectKey := ""
	if len(req.TicketID) > 36 {
		objectKey = req.TicketID[37:]
	}

	return uc.data.DB.Transaction(func(tx *gorm.DB) error {
		res := model.Resource{
			TypeKey: req.TypeKey,
			Name:    req.Name,
			OwnerID: req.OwnerID,
		}
		if err := tx.Create(&res).Error; err != nil {
			return err
		}

		ver := model.ResourceVersion{
			ResourceID: res.ID,
			VersionNum: 1,
			FilePath:   objectKey,
			FileSize:   req.Size,
			MetaData:   req.ExtraMeta,
			State:      "ACTIVE",
		}
		if err := tx.Create(&ver).Error; err != nil {
			return err
		}
		return nil
	})
}

func (uc *UseCase) GetResource(ctx context.Context, id string) (*ResourceDTO, error) {
	var r model.Resource
	if err := uc.data.DB.First(&r, "id = ?", id).Error; err != nil {
		return nil, err
	}

	var v model.ResourceVersion
	if err := uc.data.DB.Order("version_num desc").First(&v, "resource_id = ?", id).Error; err != nil {
		return nil, err
	}

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

func (uc *UseCase) ListResources(ctx context.Context, page, size int) ([]*ResourceDTO, int64, error) {
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
