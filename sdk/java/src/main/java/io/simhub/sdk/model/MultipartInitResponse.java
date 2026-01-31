package io.simhub.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.List;

@Data
public class MultipartInitResponse {
    @JsonProperty("upload_id")
    private String uploadId;
    
    @JsonProperty("key")
    private String key;
}
