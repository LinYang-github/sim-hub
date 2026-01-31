package io.simhub.sdk.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Data;
import java.util.Date;
import java.util.Map;

@Data
public class ResourceVersion {
    private String id;
    
    @JsonProperty("resource_id")
    private String resourceId;
    
    @JsonProperty("version_num")
    private Integer versionNum;
    
    private String semver;
    
    @JsonProperty("file_path")
    private String filePath;
    
    @JsonProperty("file_hash")
    private String fileHash;
    
    @JsonProperty("file_size")
    private Long fileSize;
    
    @JsonProperty("meta_data")
    private Map<String, Object> metaData;
    
    private String state;
    
    @JsonProperty("download_url")
    private String downloadUrl;
    
    @JsonProperty("created_at")
    private Date createdAt;
}
