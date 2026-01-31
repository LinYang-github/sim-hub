package io.simhub.sdk.model;

import lombok.Data;
import java.util.List;

@Data
public class ResourceListResponse {
    private List<Resource> items;
    private long total;
}
