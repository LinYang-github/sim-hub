package io.simhub.sdk;

import io.simhub.sdk.client.SimHubClient;
import java.io.InputStream;
import java.util.Random;

/**
 * Java SDK 并发分片上传压力测试工具
 */
public class StressTestApp {
    
    public static void main(String[] args) {
        String baseUrl = "http://localhost:30030";
        String token = "shp_admin_test_token"; // 请替换为实际可用的 Token
        
        // 压力测试参数
        int totalSizeMB = 50;       // 总大小 50MB
        int partSizeMB = 5;         // 分片大小 5MB
        int concurrency = 8;        // 并发数 8
        
        System.out.println("=== SimHub Java SDK 压力测试启动 ===");
        System.out.printf("部署目标: %s\n", baseUrl);
        System.out.printf("测试规模: %dMB (分片: %dMB, 并发: %d)\n", totalSizeMB, partSizeMB, concurrency);

        SimHubClient client = new SimHubClient(baseUrl, token, concurrency);
        
        // 创建一个产生伪数据的流，避免在内存中一次性分配 50MB
        long totalSizeBytes = (long) totalSizeMB * 1024 * 1024;
        long partSizeBytes = (long) partSizeMB * 1024 * 1024;
        InputStream dummyStream = new FakeDataStream(totalSizeBytes);

        long start = System.currentTimeMillis();
        try {
            client.uploadFileMultipart(
                "scenario",             // 资源类型
                "stress_test_data.bin", // 文件名
                "压测资源-" + System.currentTimeMillis(),
                "v1.0.0",
                dummyStream,
                totalSizeBytes,
                (int) partSizeBytes,
                (bytesRead, total, done) -> {
                    double progress = (double) bytesRead / total * 100;
                    System.out.printf("\r正在上传: %.2f%% [%d/%d bytes]", progress, bytesRead, total);
                    if (done) System.out.println("\n上传阶段完成！");
                }
            );

            long end = System.currentTimeMillis();
            double durationSec = (end - start) / 1000.0;
            double speedMBps = totalSizeMB / durationSec;

            System.out.println("\n=== 压测结果 ===");
            System.out.printf("总耗时: %.2f 秒\n", durationSec);
            System.out.printf("平均速度: %.2f MB/s\n", speedMBps);
            System.out.println("状态: 成功 ✅");

        } catch (Exception e) {
            System.err.println("\n压测失败 ❌");
            e.printStackTrace();
        } finally {
            client.close();
        }
    }

    /**
     * 一个不占用实际物理内存的流，用于模拟大文件
     */
    static class FakeDataStream extends InputStream {
        private final long totalSize;
        private long position = 0;
        private final Random random = new Random();

        public FakeDataStream(long totalSize) {
            this.totalSize = totalSize;
        }

        @Override
        public int read() {
            if (position >= totalSize) return -1;
            position++;
            return random.nextInt(256);
        }

        @Override
        public int read(byte[] b, int off, int len) {
            if (position >= totalSize) return -1;
            long remaining = totalSize - position;
            int toRead = (int) Math.min(len, remaining);
            // 实际上我们可以不填充数据来获得更快的本地压测速度
            // 但为了模拟真实负载，这里不做处理
            position += toRead;
            return toRead;
        }
    }
}
