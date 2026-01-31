package io.simhub.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.List;

@Data
public class MultipartInitResponse {
    @JsonProperty("ticket_id")
    private String ticketId;

    @JsonProperty("upload_id")
    private String uploadId;
    
    @JsonProperty("object_key")
    private String objectKey;
}
