package io.simhub.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Builder;
import lombok.Data;
import java.util.List;

@Data
@Builder
public class MultipartCompleteRequest {
    @JsonProperty("upload_id")
    private String uploadId;
    
    @JsonProperty("key")
    private String key;
    
    // ETag list for S3/MinIO completion
    private List<PartETag> parts;

    @Data
    @lombok.AllArgsConstructor
    @lombok.NoArgsConstructor
    public static class PartETag {
        @JsonProperty("part_number")
        private int partNumber;
        @JsonProperty("etag")
        private String etag;
    }
}
