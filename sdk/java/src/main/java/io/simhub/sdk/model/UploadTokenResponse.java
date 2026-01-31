package io.simhub.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;

@Data
public class UploadTokenResponse {
    @JsonProperty("ticket_id")
    private String ticketId;
    
    @JsonProperty("presigned_url")
    private String presignedUrl;
    
    private String method;
}
