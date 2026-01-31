from dataclasses import dataclass, field
from typing import List, Optional, Dict, Any

class SimHubError(Exception):
    """Base exception for SimHub SDK"""
    def __init__(self, message: str, status_code: Optional[int] = None):
        super().__init__(message)
        self.status_code = status_code

@dataclass
class ResourceVersion:
    id: str
    version_num: int
    semver: str
    file_size: int
    state: str
    download_url: Optional[str] = None
    meta_data: Dict[str, Any] = field(default_factory=dict)

@dataclass
class Resource:
    id: str
    type_key: str
    name: str
    owner_id: str
    scope: str
    tags: List[str]
    latest_version: Optional[ResourceVersion] = None
    created_at: Optional[str] = None

@dataclass
class ResourceListResponse:
    items: List[Resource]
    total: int
