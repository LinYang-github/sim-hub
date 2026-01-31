# 05 认证与安全 (Authentication & Security)

## 1. 认证体系 (Authentication)

SimHub 采用混合认证模式，主要针对外部访问和内部系统间通信。

### 1.1 Web/SDK Access Token
- **结构**: 数据库持久化的 Token (`shp_` 前缀)。
- **哈希存储**: 数据库仅存储 Token 的 SHA256 哈希值，确保即使数据库泄露，攻击者也无法通过令牌逆推。
- **Web 会话**: 用户登录后生成 7 天有效的专用会话令牌存储在 `localStorage` 中。

### 1.2 系统级认证
- **Auth Proxy**: 对于复杂的内网部署环境，Master 支持受信任的 X-Forwarded-User 头部注入方式进行代理认证。

## 2. 权限模型 (RBAC)

基于权限点的轻量级访问控制：
- **核心角色**:
    - `admin`: 系统超级管理员，具备 `*` 所有权限。
    - `operator`: 开发人员/资源管理人员，具备上传、更新、重命名权限。
    - `viewer`: 审计或仿真运行人员，仅具备列表和下载权限。

### 权限点清单：
- `resource:list`: 列表查看
- `resource:create`: 上传新资源
- `resource:update`: 修改元数据、重命名
- `resource:delete`: 清空资源

## 3. 安全拦截机制

### 3.1 拦截器链
API 在进入业务逻辑前会经过三层拦截：
1. **AuthMiddleware**: 验证授权头，解析用户身份。
2. **RBACMiddleware**: 校验当前用户是否具备 API 指定的权限点。
3. **OwnerShip Check**: 对于 `PRIVATE` 作用域的资源，额外校验操作者是否为该资源的所有者。

### 3.2 跨域保护
- 后端严格限制 CORS 来源。
- 前端请求拦截器自动注入 `Authorization: Bearer <token>`。
