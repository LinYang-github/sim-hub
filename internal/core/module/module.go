package module

import (
	"github.com/gin-gonic/gin"
)

// Module defines the interface for a pluggable backend module
type Module interface {
	// RegisterRoutes registers the module's routes to the Gin engine or group
	RegisterRoutes(r *gin.RouterGroup)
}

// Registry holds all registered modules
type Registry struct {
	modules []Module
}

func NewRegistry() *Registry {
	return &Registry{
		modules: make([]Module, 0),
	}
}

func (r *Registry) Register(m Module) {
	r.modules = append(r.modules, m)
}

// MapRoutes applies all module routes to the given base group (e.g. /api/v1)
func (r *Registry) MapRoutes(g *gin.RouterGroup) {
	for _, m := range r.modules {
		m.RegisterRoutes(g)
	}
}
