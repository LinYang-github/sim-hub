package io.simhub.sdk.client;

import io.simhub.sdk.ProgressCallback;
import okhttp3.MediaType;
import okhttp3.RequestBody;
import okio.BufferedSink;
import okio.Okio;
import okio.Source;

import java.io.IOException;
import java.io.InputStream;

public class ProgressRequestBody extends RequestBody {
    private final MediaType contentType;
    private final InputStream inputStream;
    private final long contentLength;
    private final ProgressCallback callback;

    public ProgressRequestBody(MediaType contentType, InputStream inputStream, long contentLength, ProgressCallback callback) {
        this.contentType = contentType;
        this.inputStream = inputStream;
        this.contentLength = contentLength;
        this.callback = callback;
    }

    @Override
    public MediaType contentType() {
        return contentType;
    }

    @Override
    public long contentLength() {
        return contentLength;
    }

    @Override
    public void writeTo(BufferedSink sink) throws IOException {
        try (Source source = Okio.source(inputStream)) {
            long totalRead = 0;
            long read;
            while ((read = source.read(sink.buffer(), 8192)) != -1) {
                totalRead += read;
                sink.flush();
                if (callback != null) {
                    callback.onProgress(totalRead, contentLength, totalRead == contentLength);
                }
            }
        }
    }
}
