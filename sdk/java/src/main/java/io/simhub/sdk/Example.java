package io.simhub.sdk;

import io.simhub.sdk.client.SimHubClient;
import io.simhub.sdk.model.Resource;
import io.simhub.sdk.model.ResourceListResponse;

import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;

public class Example {
    public static void main(String[] args) {
        // 配置参数 (请根据实际环境修改)
        String baseUrl = "http://localhost:30030";
        String token = "shp_admin_test_token"; // 示例 Token

        // 初始化客户端
        SimHubClient client = new SimHubClient(baseUrl, token);

        try {
            // 1. 简单上传示例
            System.out.println(">>> 正在演示上传...");
            String content = "Hello SimHub from Java SDK! " + System.currentTimeMillis();
            byte[] bytes = content.getBytes(StandardCharsets.UTF_8);
            
            client.uploadFileSimple(
                "documents",               // 资源类型
                "greeting.txt",            // 文件名
                "Java SDK 欢迎文档",        // 资源显示名称
                "v1.0.0",                  // 版本号
                new ByteArrayInputStream(bytes), 
                bytes.length, 
                "text/plain",
                (bytesRead, total, done) -> {
                    System.out.printf("上传进度: %d/%d (%b)\n", bytesRead, total, done);
                }
            );
            System.out.println("上传成功！");

            // 2. 查询资源列表
            System.out.println("\n>>> 正在查询资源列表...");
            ResourceListResponse response = client.listResources("documents", null, 1, 10);
            System.out.println("找到文档数量: " + response.getTotal());

            for (Resource res : response.getItems()) {
                System.out.printf("- [%s] %s\n", res.getId(), res.getName());
            }

            // 3. 详情获取与下载演示
            if (!response.getItems().isEmpty()) {
                Resource first = response.getItems().get(0);
                System.out.println("\n>>> 正在获取详情: " + first.getId());
                Resource detail = client.getResource(first.getId());
                
                if (detail.getLatestVersion() != null) {
                    System.out.println("最新版本下载地址: " + detail.getLatestVersion().getDownloadUrl());
                    
                    // 流式读取测试
                    try (InputStream in = client.downloadResource(first.getId())) {
                         // 处理输入流...
                         System.out.println("下载流连接成功。");
                    }
                }
            }

        } catch (SimHubException e) {
            System.err.println("SimHub SDK 错误: " + e.getMessage() + " (Code: " + e.getCode() + ")");
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}
