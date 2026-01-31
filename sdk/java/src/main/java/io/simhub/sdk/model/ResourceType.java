package io.simhub.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.Date;
import java.util.Map;

@Data
public class ResourceType {
    @JsonProperty("type_key")
    private String typeKey;
    
    @JsonProperty("type_name")
    private String typeName;
    
    @JsonProperty("schema_def")
    private Map<String, Object> schemaDef;
    
    @JsonProperty("category_mode")
    private String categoryMode;
    
    @JsonProperty("integration_mode")
    private String integrationMode;
    
    @JsonProperty("upload_mode")
    private String uploadMode;
    
    @JsonProperty("process_conf")
    private Map<String, Object> processConf;
    
    @JsonProperty("meta_data")
    private Map<String, Object> metaData;
    
    @JsonProperty("created_at")
    private Date createdAt;
    
    @JsonProperty("updated_at")
    private Date updatedAt;
}
