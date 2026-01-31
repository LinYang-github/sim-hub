package io.simhub.sdk;

import io.simhub.sdk.client.SimHubClient;
import okhttp3.mockwebserver.Dispatcher;
import okhttp3.mockwebserver.MockResponse;
import okhttp3.mockwebserver.MockWebServer;
import okhttp3.mockwebserver.RecordedRequest;
import org.jetbrains.annotations.NotNull;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.util.concurrent.atomic.AtomicInteger;

import static org.junit.jupiter.api.Assertions.assertEquals;

public class ParallelUploadLogicTest {
    private MockWebServer server;
    private SimHubClient client;

    @BeforeEach
    void setUp() throws IOException {
        server = new MockWebServer();
        server.start();
        client = new SimHubClient(server.url("/").toString(), "test-token", 4);
    }

    @AfterEach
    void tearDown() throws IOException {
        client.close();
        server.shutdown();
    }

    @Test
    void testParallelMultipartLogic() throws InterruptedException {
        AtomicInteger partUploadCount = new AtomicInteger(0);

        // 设置动态调度器模拟不同的 API 端点
        final Dispatcher dispatcher = new Dispatcher() {
            @NotNull
            @Override
            public MockResponse dispatch(RecordedRequest request) {
                String path = request.getPath();
                if (path == null) return new MockResponse().setResponseCode(404);

                if (path.contains("/multipart/init")) {
                    return new MockResponse()
                            .setBody("{\"upload_id\":\"u123\", \"key\":\"k123\"}")
                            .addHeader("Content-Type", "application/json");
                }
                if (path.contains("/part-url")) {
                    // 返回一个指向 MockWebServer 自身的 Put URL
                    String mockPutUrl = server.url("/mock-s3-put").toString();
                    return new MockResponse()
                            .setBody("{\"presigned_url\":\"" + mockPutUrl + "\"}")
                            .addHeader("Content-Type", "application/json");
                }
                if (path.contains("/mock-s3-put")) {
                    partUploadCount.incrementAndGet();
                    return new MockResponse().setResponseCode(200).addHeader("ETag", "\"tag-" + partUploadCount.get() + "\"");
                }
                if (path.contains("/multipart/complete")) {
                    return new MockResponse().setBody("{\"ticket_id\":\"t123\"}");
                }
                if (path.contains("/confirm")) {
                    return new MockResponse().setResponseCode(200);
                }
                return new MockResponse().setResponseCode(404);
            }
        };
        server.setDispatcher(dispatcher);

        // 模拟 20MB 数据，分 4 个 5MB 的片
        int totalSize = 20 * 1024 * 1024;
        int partSize = 5 * 1024 * 1024;
        byte[] data = new byte[totalSize];
        
        client.uploadFileMultipart("test", "test.bin", "Test", "v1.0", 
                new ByteArrayInputStream(data), totalSize, partSize, null);

        // 验证是否上传了 4 个分片
        assertEquals(4, partUploadCount.get());
    }
}
