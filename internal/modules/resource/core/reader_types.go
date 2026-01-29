package core

import (
	"context"

	"github.com/liny/sim-hub/internal/model"
)

// GetResourceTypes 获取所有资源类型定义
func (r *ResourceReader) GetResourceTypes(ctx context.Context) ([]model.ResourceType, error) {
	var types []model.ResourceType
	if err := r.data.DB.Order("sort_order asc").Find(&types).Error; err != nil {
		return nil, err
	}
	return types, nil
}
