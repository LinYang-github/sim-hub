# 前端扩展开发说明 (Frontend Extension)

SimHub 的前端界面具有高度的可配置性和可扩展性，支持动态加载预览器和元数据编辑器。

## 1. 扩展资源预览器 (Resource Previewer)

如果您新增了一种资源类型（如 `.3mx` 或特有的仿真轨迹文件），需要为其开发专用的预览插件。

### 1.1 开发步骤
1. **创建组件**: 在 `src/components/resource/previewers/` 目录下创建一个新的 Vue 3 组件。
2. **定义 Prop**: 确保组件接收 `resource` (ResourceDTO) 和 `version` (ResourceVersionDTO) 作为输入。
3. **注册组件**:
   - 打开 `src/components/resource/previewers/ResourcePreview.vue`。
   - 在 `previewComponentMap` 中添加您的组件映射关系，键名应对应 `modules.yaml` 中的 `viewer` 值。
   ```typescript
   const previewComponentMap = {
     'MySpecialViewer': markRaw(MySpecialComponent),
     // ...
   }
   ```

### 1.2 异步加载
针对体积较大的预览库（如 Cesium, JSON Editor），建议使用 `defineAsyncComponent` 进行异步导入，以减小首屏打包体积。

## 2. 扩展元数据编辑器

系统会自动根据 `modules.yaml` 中的 `schema_def` (JSON Schema) 生成基础属性编辑界面。

### 2.1 自定义组件注入 (高级)
如果标准输入框无法满足需求（例如需要地图选点获取坐标）：
1. 修改 `ResourceDetailDrawer.vue` 或其下的动态渲染子组件。
2. 根据 `type_key` 或 Schema 的特定 `ui:widget` 属性路由到您的自定义 UI 组件。

## 3. UI 样式定标
- **配色方案**: 优先使用全局定义的 CSS 变量（如 `--el-color-primary`）。
- **动效**: 推荐使用 Element Plus 的内置动画或保持与系统一致的过渡曲线，确保持续的“WOW”设计感。
