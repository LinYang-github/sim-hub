package core

type CreateResourceFromDataRequest struct {
	TypeKey      string          `json:"type_key" binding:"required"`
	CategoryID   string          `json:"category_id"`
	Name         string          `json:"name" binding:"required"`
	Data         map[string]any  `json:"data" binding:"required"` // JSON 数据本体
	SemVer       string          `json:"semver" binding:"required"`
	OwnerID      string          `json:"owner_id"`
	Scope        string          `json:"scope"`
	Tags         []string        `json:"tags"`
	Dependencies []DependencyDTO `json:"dependencies"`
	ExtraMeta    map[string]any  `json:"extra_meta"`
}
