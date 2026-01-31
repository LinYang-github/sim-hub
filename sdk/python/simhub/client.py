import os
import requests
import math
from typing import Optional, List, Callable, Dict, Any
from concurrent.futures import ThreadPoolExecutor, as_completed
from .models import Resource, ResourceVersion, ResourceListResponse, SimHubError

class SimHubClient:
    def __init__(self, base_url: str, token: str, concurrency: int = 4):
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.concurrency = concurrency
        self.session = requests.Session()
        if token:
            self.session.headers.update({"Authorization": f"Bearer {token}"})

    def _handle_response(self, response: requests.Response):
        if not response.ok:
            raise SimHubError(f"API Error ({response.status_code}): {response.text}", response.status_code)
        return response

    def list_resource_types(self) -> List[Dict[str, Any]]:
        resp = self.session.get(f"{self.base_url}/api/v1/resource-types")
        return self._handle_response(resp).json()

    def list_resources(self, type_key: str = "", page: int = 1, size: int = 20) -> ResourceListResponse:
        params = {"type_key": type_key, "page": page, "size": size}
        resp = self.session.get(f"{self.base_url}/api/v1/resources", params=params)
        data = self._handle_response(resp).json()
        
        items = []
        for item in data.get("items", []):
            lv_data = item.get("latest_version")
            lv = ResourceVersion(**lv_data) if lv_data else None
            items.append(Resource(
                id=item["id"],
                type_key=item["type_key"],
                name=item["name"],
                owner_id=item["owner_id"],
                scope=item["scope"],
                tags=item.get("tags", []),
                latest_version=lv,
                created_at=item.get("created_at")
            ))
        return ResourceListResponse(items=items, total=data.get("total", 0))

    def get_resource(self, resource_id: str) -> Resource:
        resp = self.session.get(f"{self.base_url}/api/v1/resources/{resource_id}")
        item = self._handle_response(resp).json()
        lv_data = item.get("latest_version")
        lv = ResourceVersion(**lv_data) if lv_data else None
        return Resource(
            id=item["id"],
            type_key=item["type_key"],
            name=item["name"],
            owner_id=item["owner_id"],
            scope=item["scope"],
            tags=item.get("tags", []),
            latest_version=lv,
            created_at=item.get("created_at")
        )

    def upload_file_simple(self, type_key: str, file_path: str, name: str, semver: str = "1.0.0", 
                          content_type: str = "application/octet-stream",
                          progress_callback: Optional[Callable[[int, int], None]] = None):
        file_size = os.path.getsize(file_path)
        
        # 1. Get Token
        token_req = {
            "resource_type": type_key,
            "filename": os.path.basename(file_path),
            "size": file_size
        }
        resp = self.session.post(f"{self.base_url}/api/v1/integration/upload/token", json=token_req)
        token_data = self._handle_response(resp).json()
        
        # 2. Upload to S3
        with open(file_path, "rb") as f:
            # Note: requests doesn't natively support easy progress for basic files without wrapping,
            # but for simple upload we can just send it.
            # If progress is needed, we could use a custom generator or 'rebound' approach.
            put_resp = requests.put(token_data["presigned_url"], data=f, headers={"Content-Type": content_type})
            if not put_resp.ok:
                raise SimHubError(f"Upload failed: {put_resp.text}")
        
        # 3. Confirm
        confirm_req = {
            "ticket_id": token_data["ticket_id"],
            "name": name,
            "semver": semver,
            "type_key": type_key
        }
        self._handle_response(self.session.post(f"{self.base_url}/api/v1/integration/upload/confirm", json=confirm_req))

    def upload_file_multipart(self, type_key: str, file_path: str, name: str, semver: str = "1.0.0", 
                             part_size: int = 5 * 1024 * 1024, 
                             progress_callback: Optional[Callable[[int, int], None]] = None):
        file_size = os.path.getsize(file_path)
        part_count = math.ceil(file_size / part_size)

        # 1. Init
        init_req = {
            "resource_type": type_key,
            "filename": os.path.basename(file_path),
            "part_count": part_count
        }
        resp = self.session.post(f"{self.base_url}/api/v1/integration/upload/multipart/init", json=init_req)
        init_data = self._handle_response(resp).json()
        upload_id = init_data["upload_id"]
        object_key = init_data["object_key"]

        # 2. Parallel Upload
        etags = [None] * part_count
        uploaded_bytes = 0

        def upload_part(part_num):
            offset = (part_num - 1) * part_size
            current_part_size = min(part_size, file_size - offset)
            
            # Get Part URL
            payload = {"upload_id": upload_id, "ticket_id": init_data["ticket_id"], "part_number": part_num}
            u_resp = self.session.post(f"{self.base_url}/api/v1/integration/upload/multipart/part-url", json=payload)
            presigned_url = self._handle_response(u_resp).json()["url"]

            # Upload to S3
            with open(file_path, "rb") as f:
                f.seek(offset)
                data = f.read(current_part_size)
                put_resp = requests.put(presigned_url, data=data)
                if not put_resp.ok:
                    raise SimHubError(f"Part {part_num} upload failed: {put_resp.text}")
                etag = put_resp.headers.get("ETag", "").strip('"')
                return part_num, etag, current_part_size

        with ThreadPoolExecutor(max_workers=self.concurrency) as executor:
            futures = [executor.submit(upload_part, i + 1) for i in range(part_count)]
            for future in as_completed(futures):
                p_num, p_etag, p_size = future.result()
                etags[p_num - 1] = {"part_number": p_num, "etag": p_etag}
                uploaded_bytes += p_size
                if progress_callback:
                    progress_callback(uploaded_bytes, file_size)

        # 3. Complete
        complete_req = {
            "upload_id": upload_id,
            "object_key": object_key,
            "ticket_id": init_data["ticket_id"],
            "parts": etags,
            "type_key": type_key,
            "name": name,
            "semver": semver,
            "owner_id": "admin",
            "scope": "public"
        }
        resp = self.session.post(f"{self.base_url}/api/v1/integration/upload/multipart/complete", json=complete_req)
        self._handle_response(resp)

    def download_file(self, resource_id: str, target_path: str, progress_callback: Optional[Callable[[int, int], None]] = None):
        res = self.get_resource(resource_id)
        if not res.latest_version or not res.latest_version.download_url:
            raise SimHubError("Resource has no active version or download URL")
        
        with self.session.get(res.latest_version.download_url, stream=True) as r:
            self._handle_response(r)
            total_size = int(r.headers.get('content-length', 0))
            downloaded = 0
            with open(target_path, 'wb') as f:
                for chunk in r.iter_content(chunk_size=8192):
                    f.write(chunk)
                    downloaded += len(chunk)
                    if progress_callback:
                        progress_callback(downloaded, total_size)
