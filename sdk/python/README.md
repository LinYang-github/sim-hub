# SimHub SDK for Python

SimHub 的 Python 版本 SDK，专为数据分析、自动化流水线与仿真控制设计。

## 特性
- **简洁高效**: 基于 `requests` 封装，接口符合 Python 之禅。
- **并发加速**: 内置 `ThreadPoolExecutor` 实现多线程分片上传。
- **流式下载**: 支持超大文件流式下载，内存占用极低。
- **现代化**: 使用 `dataclasses` 提供完美的 IDE 类型补全。

## 安装

```bash
cd sdk/python
pip install .
```

## 快速上手

```python
from simhub import SimHubClient

client = SimHubClient("http://localhost:30030", "YOUR_TOKEN")

# 并发上传模型
client.upload_file_multipart(
    type_key="terrain",
    file_path="map.bin",
    name="高精度地形",
    progress_callback=lambda curr, total: print(f"进度: {curr/total:.1%}")
)
```

## 测试

```bash
cd sdk/python
pytest
```
