package service

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liny/sim-hub/internal/biz"
)

type ResourceService struct {
	uc *biz.ResourceUseCase
}

func NewResourceService(uc *biz.ResourceUseCase) *ResourceService {
	return &ResourceService{uc: uc}
}

// ApplyUploadToken handles POST /api/v1/integration/upload/token
func (s *ResourceService) ApplyUploadToken(c *gin.Context) {
	var req struct {
		Filename string `json:"filename" binding:"required"`
		TypeKey  string `json:"type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ticket, err := s.uc.RequestUploadToken(c.Request.Context(), req.Filename, req.TypeKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

// ConfirmUpload handles POST /api/v1/integration/upload/confirm
func (s *ResourceService) ConfirmUpload(c *gin.Context) {
	var req biz.ConfirmUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.uc.ConfirmUpload(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Processing started"})
}

// GetResource handles GET /api/v1/resources/:id
func (s *ResourceService) GetResource(c *gin.Context) {
	id := c.Param("id")
	res, err := s.uc.GetResource(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
		return
	}
	c.JSON(http.StatusOK, res)
}

// ListResources handles GET /api/v1/resources
func (s *ResourceService) ListResources(c *gin.Context) {
	// Pagination (Simple)
	page := 1
	size := 20
	// Parse query params if needed...

	list, total, err := s.uc.ListResources(c.Request.Context(), page, size)
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
