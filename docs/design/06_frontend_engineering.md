# 06 前端工程体系 (Frontend Engineering)

## 1. 技术栈与架构
基于 **Vue 3 (Composition API) + Vite + Element Plus** 构建的单页应用。

## 2. 设计系统与美学
SimHub 前端追求极致的“WOW”感和专业感。
- **暗色主题 (Obsidian Theme)**: 深色背景配以高对比度的主题色，符合仿真/工程软件审美。
- **视觉层级**:
    - 磨砂玻璃 (Glassmorphism): 导航栏和抽屉面板使用 `backdrop-filter: blur`。
    - 渐变过渡 (Vibrant States): 状态标签使用柔和的背景渐变区分资源状态（Active/Processing）。

## 3. 核心机制

### 3.1 动态资源驱动 (Module Manager)
前端完全不写死资源类型。资源列表、分类树和元数据编辑器均通过后端下发的 `ResourceType` 动态渲染。
- **CategorySidebar**: 根据 `CategoryMode` 自动切换多级树或平面列表。
- **ResourcePreview**: 一个通用预览组件，根据资源定义的 `viewer` 类型（GLB, PDF, Image等）异步加载特定预览插件。

### 3.2 离线处理能力
- **JSZip Browser-Side**: 在上传大型成果包之前，前端直接在浏览器执行目录树解析和预览。
- **Multipart Uploading**: 利用前端流式上传技术，分片推送到 MinIO。

## 4. 性能优化 (Perceived Performance)
- **异步组件**: 复杂的场景预览器（如 Cesium, JSON Tree）采用 `defineAsyncComponent` 按需加载。
- **状态管理**: 尽量使用 `composables` (setup functions) 而非全局 Vuex 类库，使数据流向更清晰、打包体积更小。
- **Skeleton Screens**: 资源列表加载时展示骨架屏，减少由于网络抖动带来的闪烁感。

## 5. 交互设计
- **Spotlight Search (Ctrl/Cmd+K)**: 沉浸式全局搜索中心。
- **Resource Detail Drawer**: 通过右侧抽屉展示元数据，不离开主列表，保持工作流的连贯性。
