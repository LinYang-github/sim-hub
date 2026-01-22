package resource

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liny/sim-hub/internal/core/module"
	"github.com/liny/sim-hub/internal/data"
	"github.com/liny/sim-hub/internal/modules/resource/core"
	"github.com/liny/sim-hub/pkg/sts"
)

// Module implements module.Module
type Module struct {
	uc *core.UseCase
}

func NewModule(d *data.Data, tv *sts.TokenVendor, bucket string) module.Module {
	return &Module{
		uc: core.NewUseCase(d, tv, bucket),
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

func (m *Module) ApplyUploadToken(c *gin.Context) {
	var req core.ApplyUploadTokenRequest
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
	var req core.ConfirmUploadRequest
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
	page := 1  // TODO: parse page from query
	size := 20 // TODO: parse size from query
	typeKey := c.Query("type")

	list, total, err := m.uc.ListResources(c.Request.Context(), typeKey, page, size)
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
