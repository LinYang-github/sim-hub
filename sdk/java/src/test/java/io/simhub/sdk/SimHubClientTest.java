package io.simhub.sdk;

import io.simhub.sdk.client.SimHubClient;
import io.simhub.sdk.model.Resource;
import io.simhub.sdk.model.ResourceListResponse;
import okhttp3.mockwebserver.MockResponse;
import okhttp3.mockwebserver.MockWebServer;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.io.IOException;

import static org.junit.jupiter.api.Assertions.*;

public class SimHubClientTest {
    private MockWebServer server;
    private SimHubClient client;

    @BeforeEach
    void setUp() throws IOException {
        server = new MockWebServer();
        server.start();
        client = new SimHubClient(server.url("/").toString(), "test-token");
    }

    @AfterEach
    void tearDown() throws IOException {
        server.shutdown();
    }

    @Test
    void testListResources() {
        String json = "{\"items\":[{\"id\":\"1\",\"name\":\"test-res\"}],\"total\":1}";
        server.enqueue(new MockResponse()
                .setBody(json)
                .addHeader("Content-Type", "application/json"));

        ResourceListResponse response = client.listResources("scenario", null, 1, 10);
        
        assertEquals(1, response.getTotal());
        assertEquals("test-res", response.getItems().get(0).getName());
    }

    @Test
    void testGetResource() {
        String json = "{\"id\":\"123\",\"name\":\"details\"}";
        server.enqueue(new MockResponse()
                .setBody(json)
                .addHeader("Content-Type", "application/json"));

        Resource resource = client.getResource("123");
        
        assertEquals("123", resource.getId());
        assertEquals("details", resource.getName());
    }

    @Test
    void testApiError() {
        server.enqueue(new MockResponse().setResponseCode(401).setBody("Unauthorized"));

        assertThrows(SimHubException.class, () -> client.getResource("123"));
    }
}
