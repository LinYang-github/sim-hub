from simhub.client import SimHubClient
import os
import time

def main():
    base_url = "http://localhost:30030"
    token = "shp_0a1292f15c64595eda39562274db6900eb3a29c8d4a46e765899a171bf0c197c"
    
    client = SimHubClient(base_url, token, concurrency=8)
    
    # 1. 列表展示
    print(">>> 正在拉取资源列表...")
    resp = client.list_resources(type_key="scenario")
    print(f"总计资源: {resp.total}")
    for res in resp.items:
        print(f"- [{res.id}] {res.name} (版本: {res.latest_version.semver if res.latest_version else 'N/A'})")

    # 2. 分片上传逻辑演示
    test_file = "python_test_large.bin"
    file_size_mb = 12
    print(f"\n>>> 准备测试文件 ({file_size_mb}MB)...")
    with open(test_file, "wb") as f:
        f.write(os.urandom(file_size_mb * 1024 * 1024))
    
    try:
        print(">>> 开始并发分片上传...")
        start_time = time.time()
        client.upload_file_multipart(
            type_key="scenario",
            file_path=test_file,
            name="Python 并发测试资源",
            semver="1.0.0",
            progress_callback=lambda current, total: print(f"\r进度: {current/total*100:.2f}% ({current}/{total})", end="")
        )
        duration = time.time() - start_time
        print(f"\n上传成功！耗时: {duration:.2f}秒, 速度: {file_size_mb/duration:.2f} MB/s")
        
    finally:
        if os.path.exists(test_file):
            os.remove(test_file)

if __name__ == "__main__":
    main()
