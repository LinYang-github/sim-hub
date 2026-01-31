# SimHub SDK for Java

SimHub 的 Java 版本 SDK，旨在为基于 JVM 的仿真程序提供便捷的资产访问能力。

## 核心特性
- 基于 OKHttp 4.0 的轻量级异步/同步请求。
- 自动处理 Token 鉴权。
- 支持资源列表查询、关键词搜索、详情获取。
- 支持资源的直接流式下载。
- 对 Jackson 序列化友好，支持自定义 MetaData 处理。

## 安装方式

### Maven
将本项目作为依赖添加到您的 `pom.xml` 中（目前建议通过源码引入或本地 Maven 仓库安装）：

```xml
<dependency>
    <groupId>io.simhub</groupId>
    <artifactId>simhub-sdk-java</artifactId>
    <version>1.0.0</version>
</dependency>
```

### 依赖库
- OkHttp 4.x
- Jackson Databind 2.x
- Project Lombok (可选，编译时需要)

## 快速上手

```java
import io.simhub.sdk.client.SimHubClient;
import io.simhub.sdk.model.Resource;

// 1. 初始化
SimHubClient client = new SimHubClient("http://simhub-api:30030", "shp_xxxx");

// 2. 搜索资源
ResourceListResponse res = client.listResources("scenario", "高架桥", 1, 10);

// 3. 下载资源
InputStream in = client.downloadResource("resource-uuid");
// 将流保存到本地或载入内存...
```

## 异常处理
SDK 抛出统一的 `SimHubException`，包含错误信息及可选的 HTTP 状态码。

```java
try {
    client.getResource("invalid-id");
} catch (SimHubException e) {
    if (e.getCode() == 404) {
        System.out.println("资源不存在");
    }
}
```
