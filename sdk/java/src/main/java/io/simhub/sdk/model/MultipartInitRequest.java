package io.simhub.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Builder;
import lombok.Data;

@Data
@Builder
public class MultipartInitRequest {
    @JsonProperty("resource_type")
    private String resourceType;
    
    private String filename;
    
    @JsonProperty("part_count")
    private int partCount;
}
