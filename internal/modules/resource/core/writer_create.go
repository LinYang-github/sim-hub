package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateResourceFromData 处理在线表单创建资源的逻辑
func (w *ResourceWriter) CreateResourceFromData(ctx context.Context, req CreateResourceFromDataRequest) (*ResourceDTO, error) {
	// 1. 将数据序列化为 JSON
	dataBytes, err := json.MarshalIndent(req.Data, "", "  ") // Indent for readability
	if err != nil {
		return nil, fmt.Errorf("serialize data failed: %w", err)
	}

	// 2. 生成对象存储路径
	// 规则: resources/<type>/<uuid>/data_<ts>.json
	resourceUUID := uuid.New().String()
	objectKey := fmt.Sprintf("resources/%s/%s/data_%d.json", req.TypeKey, resourceUUID, time.Now().Unix())

	// 3. 上传到 Blob Store
	if err := w.store.Put(ctx, w.bucket, objectKey, bytes.NewReader(dataBytes), int64(len(dataBytes)), "application/json"); err != nil {
		return nil, fmt.Errorf("upload to storage failed: %w", err)
	}

	// 4. 利用现有的 CreateResourceAndVersion 逻辑写入数据库
	// 注意：我们将 Resource ID 提前生成了，但 CreateResourceAndVersion 内部可能会新建资源
	// 为了复用，我们实际上不需要手动生成 ResourceID，除非我们想控制 path。
	// 这里为了简单，我们让 CreateResourceAndVersion 处理 DB 事务，我们只需要提供 upload 好的 objectKey。
	// 但是 CreateResourceAndVersion 内部并不假设 ID，而是通过 name/category/type/owner 查找或创建。

	err = w.data.DB.Transaction(func(tx *gorm.DB) error {
		// 复用核心写入逻辑
		// 注意: data 被同时视为 meta_data 的一部分，方便后续索引
		// 为了不污染标准 meta_data，可以将 raw data 放入一个字段，或者直接 merge。
		// 这里 req.Data 已经是用户填写的业务字段，应该直接作为 meta_data 存储

		mergedMeta := make(map[string]any)
		if req.ExtraMeta != nil {
			for k, v := range req.ExtraMeta {
				mergedMeta[k] = v
			}
		}
		// 将表单数据也存入 metadata，便于搜索
		for k, v := range req.Data {
			mergedMeta[k] = v
		}

		return w.CreateResourceAndVersion(
			tx,
			req.TypeKey,
			req.CategoryID,
			req.Name,
			req.OwnerID,
			req.Scope,
			objectKey,
			int64(len(dataBytes)),
			req.Tags,
			req.SemVer,
			req.Dependencies,
			mergedMeta,
		)
	})

	if err != nil {
		return nil, err
	}

	// 5. 由于 CreateResourceAndVersion 没有返回 ID，我们需要查询回来返回给前端 (Optional)
	// 这里为了简化，只返回 nil error，前端列表会刷新。
	// 但 ResourceDTO 还是建议构建一个简单的
	return &ResourceDTO{
		Name:      req.Name,
		TypeKey:   req.TypeKey,
		CreatedAt: time.Now(),
	}, nil
}
