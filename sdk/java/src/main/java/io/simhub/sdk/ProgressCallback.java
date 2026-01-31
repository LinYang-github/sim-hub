package io.simhub.sdk;

public interface ProgressCallback {
    void onProgress(long bytesRead, long contentLength, boolean done);
}
