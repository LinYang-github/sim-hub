package io.simhub.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.Date;
import java.util.List;

@Data
public class Resource {
    private String id;
    
    @JsonProperty("type_key")
    private String typeKey;
    
    @JsonProperty("resource_type")
    private ResourceType resourceType;
    
    @JsonProperty("category_id")
    private String categoryId;
    
    private Category category;
    
    private String name;
    
    @JsonProperty("owner_id")
    private String ownerId;
    
    private String scope;
    
    private List<String> tags;
    
    @JsonProperty("latest_version_id")
    private String latestVersionId;
    
    @JsonProperty("latest_version")
    private ResourceVersion latestVersion;
    
    @JsonProperty("created_at")
    private Date createdAt;
    
    @JsonProperty("updated_at")
    private Date updatedAt;
}
