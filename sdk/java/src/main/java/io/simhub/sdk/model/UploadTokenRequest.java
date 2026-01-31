package io.simhub.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Builder;
import lombok.Data;

@Data
@Builder
public class UploadTokenRequest {
    @JsonProperty("resource_type")
    private String resourceType;
    
    private String filename;
    
    private Long size;
    
    private String checksum;
    
    @Builder.Default
    private String mode = "presigned";
}
