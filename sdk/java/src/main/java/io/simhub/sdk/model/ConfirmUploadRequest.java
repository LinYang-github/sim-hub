package io.simhub.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Builder;
import lombok.Data;
import java.util.List;
import java.util.Map;

@Data
@Builder
public class ConfirmUploadRequest {
    @JsonProperty("ticket_id")
    private String ticketId;
    
    private String name;
    
    private List<String> tags;
    
    private String semver;
    
    @JsonProperty("category_id")
    private String categoryId;
    
    @JsonProperty("meta_data")
    private Map<String, Object> metaData;
}
