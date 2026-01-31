# SimHub SDK for Go

SimHub 的 Go 语言版本 SDK，支持简单的资源管理与高效的并发上传。

## 特性
- **原生并发**: 利用 Go 的 goroutine 轻松实现分片并发上传。
- **内存安全**: 仅在需要时读取文件分片，支持 GB 级大文件。
- **Context 支持**: 完美集成 Go 的 `context` 标准库用于超时与取消。
- **零依赖**: 仅使用标准库实现，体积轻量，集成简单。

## 安装

```bash
go get io.simhub/sdk/go
```

## 快速上手

```go
import "io.simhub/sdk/go"

client := simhub.NewClient("http://localhost:30030", "your_token")
client.SetConcurrency(8) // 设置分片上传并发度

// 上传文件
err := client.UploadFileMultipart(ctx, "scenario", "test.zip", "Scenario A", "1.0.0", nil)

// 列出资源
res, err := client.ListResources(ctx, "scenario", 1, 10)
```

## 测试

```bash
cd sdk/go
go test -v .
```
