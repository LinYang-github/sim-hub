package io.simhub.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Builder;
import lombok.Data;
import java.util.List;

@Data
@Builder
public class MultipartCompleteRequest {
    @JsonProperty("ticket_id")
    private String ticketId;

    @JsonProperty("upload_id")
    private String uploadId;
    
    @JsonProperty("object_key")
    private String objectKey;
    
    // ETag list for S3/MinIO completion
    private List<PartETag> parts;

    // Resource metadata for registration after completion
    @JsonProperty("type_key")
    private String typeKey;
    private String name;
    private String semver;
    @JsonProperty("owner_id")
    private String ownerId;
    private String scope;

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
