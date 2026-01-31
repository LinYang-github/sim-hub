package io.simhub.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.Date;

@Data
public class Category {
    private String id;
    
    @JsonProperty("type_key")
    private String typeKey;
    
    private String name;
    
    @JsonProperty("parent_id")
    private String parentId;
    
    @JsonProperty("created_at")
    private Date createdAt;
    
    @JsonProperty("updated_at")
    private Date updatedAt;
}
