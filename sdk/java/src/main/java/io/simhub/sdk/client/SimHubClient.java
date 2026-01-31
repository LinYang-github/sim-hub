package io.simhub.sdk.client;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import io.simhub.sdk.ProgressCallback;
import io.simhub.sdk.SimHubException;
import io.simhub.sdk.model.*;
import okhttp3.*;

import java.io.IOException;
import java.io.InputStream;
import java.util.Map;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicLong;

public class SimHubClient {
    private final String baseUrl;
    private final String token;
    private final OkHttpClient httpClient;
    private final ObjectMapper objectMapper;
    private final ExecutorService executorService;
    private int concurrency = 4;

    public SimHubClient(String baseUrl, String token) {
        this(baseUrl, token, 4);
    }

    public SimHubClient(String baseUrl, String token, int concurrency) {
        this.baseUrl = baseUrl.endsWith("/") ? baseUrl.substring(0, baseUrl.length() - 1) : baseUrl;
        this.token = token;
        this.concurrency = concurrency;
        this.httpClient = new OkHttpClient.Builder()
                .connectTimeout(10, TimeUnit.SECONDS)
                .readTimeout(30, TimeUnit.SECONDS)
                .connectionPool(new ConnectionPool(concurrency, 5, TimeUnit.MINUTES))
                .build();
        this.objectMapper = new ObjectMapper()
                .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        this.executorService = Executors.newFixedThreadPool(concurrency);
    }

    /**
     * 关闭 SDK 释放资源 (如线程池)
     */
    public void close() {
        executorService.shutdown();
        try {
            if (!executorService.awaitTermination(5, TimeUnit.SECONDS)) {
                executorService.shutdownNow();
            }
        } catch (InterruptedException e) {
            executorService.shutdownNow();
        }
    }

    public java.util.List<ResourceType> listResourceTypes() {
        String url = baseUrl + "/api/v1/resource-types";
        Request request = new Request.Builder()
                .url(url)
                .header("Authorization", "Bearer " + token)
                .get()
                .build();

        try (Response response = httpClient.newCall(request).execute()) {
            handleErrorResponse(response);
            return objectMapper.readValue(response.body().string(), new com.fasterxml.jackson.core.type.TypeReference<java.util.List<ResourceType>>() {});
        } catch (IOException e) {
            throw new SimHubException("Failed to list resource types: " + e.getMessage());
        }
    }

    public ResourceListResponse listResources(String typeKey, String keyword, Integer page, Integer size) {
        StringBuilder urlBuilder = new StringBuilder(baseUrl)
                .append("/api/v1/resources?page=")
                .append(page != null ? page : 1)
                .append("&size=")
                .append(size != null ? size : 20);

        if (typeKey != null && !typeKey.isEmpty()) {
            urlBuilder.append("&type_key=").append(typeKey);
        }
        if (keyword != null && !keyword.isEmpty()) {
            urlBuilder.append("&keyword=").append(keyword);
        }

        Request request = new Request.Builder()
                .url(urlBuilder.toString())
                .header("Authorization", "Bearer " + token)
                .get()
                .build();

        try (Response response = httpClient.newCall(request).execute()) {
            handleErrorResponse(response);
            return objectMapper.readValue(response.body().string(), ResourceListResponse.class);
        } catch (IOException e) {
            throw new SimHubException("Failed to list resources: " + e.getMessage());
        }
    }

    public Resource getResource(String id) {
        String url = baseUrl + "/api/v1/resources/" + id;
        Request request = new Request.Builder()
                .url(url)
                .header("Authorization", "Bearer " + token)
                .get()
                .build();

        try (Response response = httpClient.newCall(request).execute()) {
            handleErrorResponse(response);
            return objectMapper.readValue(response.body().string(), Resource.class);
        } catch (IOException e) {
            throw new SimHubException("Failed to get resource: " + e.getMessage());
        }
    }

    public void uploadFileSimple(String resourceType, String filename, String name, String semver, InputStream inputStream, long size, String contentType, ProgressCallback callback) {
        UploadTokenRequest tokenReq = UploadTokenRequest.builder()
                .resourceType(resourceType)
                .filename(filename)
                .size(size)
                .build();

        try {
            UploadTokenResponse tokenResp;
            try (Response response = postJson(baseUrl + "/api/v1/integration/upload/token", tokenReq)) {
                handleErrorResponse(response);
                tokenResp = objectMapper.readValue(response.body().string(), UploadTokenResponse.class);
            }

            Request uploadReq = new Request.Builder()
                    .url(tokenResp.getPresignedUrl())
                    .put(new ProgressRequestBody(MediaType.parse(contentType), inputStream, size, callback))
                    .build();

            try (Response response = httpClient.newCall(uploadReq).execute()) {
                handleErrorResponse(response);
            }

            ConfirmUploadRequest confirmReq = ConfirmUploadRequest.builder()
                    .ticketId(tokenResp.getTicketId())
                    .name(name)
                    .semver(semver)
                    .build();

            try (Response response = postJson(baseUrl + "/api/v1/integration/upload/confirm", confirmReq)) {
                handleErrorResponse(response);
            }

        } catch (IOException e) {
            throw new SimHubException("Failed to upload file: " + e.getMessage());
        }
    }

    /**
     * 调优后的并发分片上传大文件
     */
    public void uploadFileMultipart(String resourceType, String filename, String name, String semver, InputStream inputStream, long size, int partSize, ProgressCallback callback) {
        try {
            // 1. 初始化
            int partCount = (int) Math.ceil((double) size / partSize);
            MultipartInitRequest initReq = MultipartInitRequest.builder()
                    .resourceType(resourceType)
                    .filename(filename)
                    .partCount(partCount)
                    .build();

            MultipartInitResponse initResp;
            try (Response response = postJson(baseUrl + "/api/v1/integration/upload/multipart/init", initReq)) {
                handleErrorResponse(response);
                initResp = objectMapper.readValue(response.body().string(), MultipartInitResponse.class);
            }

            // 2. 并发上传分片
            // 使用 Semaphore 限制内存占用，避免同时读取太多分片到内存
            Semaphore semaphore = new Semaphore(concurrency);
            CompletionService<MultipartCompleteRequest.PartETag> completionService = new ExecutorCompletionService<>(executorService);
            AtomicLong totalBytesUploaded = new AtomicLong(0);

            for (int i = 1; i <= partCount; i++) {
                int currentPartNumber = i;
                int currentPartSize = (int) Math.min(partSize, size - (long)(i-1) * partSize);
                
                // 顺序读取 InputSteam 到缓冲区
                byte[] buffer = new byte[currentPartSize];
                int read = 0;
                while (read < currentPartSize) {
                    int n = inputStream.read(buffer, read, currentPartSize - read);
                    if (n == -1) break;
                    read += n;
                }

                // 提交任务前先申请信号量
                semaphore.acquire();
                
                final String uploadId = initResp.getUploadId();
                final String key = initResp.getKey();

                completionService.submit(() -> {
                    try {
                        // A. 获取分片上传 URL
                        String partUrlQuery = String.format("%s/api/v1/integration/upload/multipart/part-url?upload_id=%s&key=%s&part_number=%d",
                                baseUrl, uploadId, key, currentPartNumber);
                        
                        String presignedUrl;
                        try (Response resp = getRequest(partUrlQuery)) {
                            handleErrorResponse(resp);
                            @SuppressWarnings("unchecked")
                            Map<String, String> urlMap = objectMapper.readValue(resp.body().string(), Map.class);
                            presignedUrl = urlMap.get("presigned_url");
                        }

                        // B. 执行上传
                        Request putReq = new Request.Builder()
                                .url(presignedUrl)
                                .put(RequestBody.create(buffer, null))
                                .build();

                        try (Response resp = httpClient.newCall(putReq).execute()) {
                            handleErrorResponse(resp);
                            String etag = resp.header("ETag");
                            
                            // 更新进度
                            long uploaded = totalBytesUploaded.addAndGet(currentPartSize);
                            if (callback != null) {
                                callback.onProgress(uploaded, size, uploaded == size);
                            }
                            
                            return new MultipartCompleteRequest.PartETag(currentPartNumber, etag);
                        }
                    } finally {
                        semaphore.release();
                    }
                });
            }

            // 3. 等待所有任务完成并收集 ETag
            MultipartCompleteRequest.PartETag[] partETags = new MultipartCompleteRequest.PartETag[partCount];
            for (int i = 0; i < partCount; i++) {
                try {
                    MultipartCompleteRequest.PartETag tag = completionService.take().get();
                    partETags[tag.getPartNumber() - 1] = tag;
                } catch (InterruptedException | ExecutionException e) {
                    throw new SimHubException("Part upload interrupted or failed: " + e.getMessage());
                }
            }

            // 4. 完成合并
            MultipartCompleteRequest completeReq = MultipartCompleteRequest.builder()
                    .uploadId(initResp.getUploadId())
                    .key(initResp.getKey())
                    .parts(java.util.Arrays.asList(partETags))
                    .build();

            String ticketId;
            try (Response response = postJson(baseUrl + "/api/v1/integration/upload/multipart/complete", completeReq)) {
                handleErrorResponse(response);
                @SuppressWarnings("unchecked")
                Map<String, String> resMap = objectMapper.readValue(response.body().string(), Map.class);
                ticketId = resMap.get("ticket_id");
            }

            // 5. 最终确认
            ConfirmUploadRequest confirmReq = ConfirmUploadRequest.builder()
                    .ticketId(ticketId)
                    .name(name)
                    .semver(semver)
                    .build();

            try (Response response = postJson(baseUrl + "/api/v1/integration/upload/confirm", confirmReq)) {
                handleErrorResponse(response);
            }

        } catch (IOException | InterruptedException e) {
            throw new SimHubException("Multipart upload failed: " + e.getMessage());
        }
    }

    public InputStream downloadResource(String id) {
        Resource resource = getResource(id);
        if (resource.getLatestVersion() == null || resource.getLatestVersion().getDownloadUrl() == null) {
            throw new SimHubException("Resource has no active version or download URL");
        }

        Request request = new Request.Builder()
                .url(resource.getLatestVersion().getDownloadUrl())
                .get()
                .build();

        try {
            Response response = httpClient.newCall(request).execute();
            handleErrorResponse(response);
            return response.body().byteStream();
        } catch (IOException e) {
            throw new SimHubException("Failed to download resource: " + e.getMessage());
        }
    }

    private Response postJson(String url, Object body) throws IOException {
        String json = objectMapper.writeValueAsString(body);
        Request request = new Request.Builder()
                .url(url)
                .header("Authorization", "Bearer " + token)
                .post(RequestBody.create(json, MediaType.parse("application/json")))
                .build();
        return httpClient.newCall(request).execute();
    }

    private Response getRequest(String url) throws IOException {
        Request request = new Request.Builder()
                .url(url)
                .header("Authorization", "Bearer " + token)
                .get()
                .build();
        return httpClient.newCall(request).execute();
    }

    private void handleErrorResponse(Response response) throws IOException {
        if (!response.isSuccessful()) {
            String errorBody = response.body() != null ? response.body().string() : "No error body";
            throw new SimHubException("API Error (Status " + response.code() + "): " + errorBody, response.code());
        }
    }
}
