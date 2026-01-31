#ifndef SIMHUB_SDK_HPP
#define SIMHUB_SDK_HPP

#include <stddef.h>
#include <stdint.h>
#include <string>
#include <vector>
#include <map>
#include <set>
#include <memory>
#include <functional>
#include <future>

/**
 * SIMHUB ABI STABLE SDK
 * 
 * This header provides a stable ABI while maintaining C++ ease of use.
 * STL containers are used in the public interface for convenience, 
 * but internal implementation is strictly decoupled via PImpl.
 */

namespace simhub {

enum class ErrorCode {
    Success = 0,
    NetworkError,
    ServerError,
    InvalidParam,
    FileSystemError,
    StorageError,
    Unknown
};

// --- ABI Safe Result ---
template <typename T>
struct Result {
    T value;
    ErrorCode code = ErrorCode::Success;
    std::string message;

    bool ok() const { return code == ErrorCode::Success; }
    static Result<T> Success(T val) { return {val, ErrorCode::Success, ""}; }
    static Result<T> Fail(ErrorCode c, std::string msg) { return {T(), c, msg}; }
};

using Status = Result<bool>;

// --- ABI Safe Types ---

struct ResourceVersion {
    int version_num;
    long long file_size;
    std::string download_url;
    std::string semver;
    std::string state;
    std::map<std::string, std::string> meta_data;
};

struct Resource {
    std::string id;
    std::string type_key;
    std::string category_id;
    std::string name;
    std::string owner_id;
    std::string scope;
    std::vector<std::string> tags;
    ResourceVersion latest_version;
    std::string created_at;
};

struct Category {
    std::string id;
    std::string type_key;
    std::string name;
    std::string parent_id;
};

struct Dependency {
    std::string target_resource_id;
    std::string constraint;
};

struct UploadTokenRequest {
    std::string resource_type;
    std::string filename;
    long long size;
    std::string checksum;
    std::string mode; // "presigned" or "sts"
};

struct STSCredentials {
    std::string access_key;
    std::string secret_key;
    std::string session_token;
    std::string expiration;
};

struct UploadTicket {
    std::string ticket_id;
    std::string presigned_url;
    STSCredentials credentials;
    std::string bucket;
    std::string object_key;
    bool has_credentials;
};

// --- Client Interface ---
class ClientImpl;

class Client {
public:
    explicit Client(const std::string& baseUrl);
    void setToken(const std::string& token);
    const std::string& getBaseUrl() const;
    ~Client();

    Client(const Client&) = delete;
    Client& operator=(const Client&) = delete;

    static void GlobalInit();
    static void GlobalCleanup();

    // Discovery (Sync)
    Result<Resource> getResource(const std::string& id);
    Result<std::vector<Resource>> listResources(const std::string& typeKey = "", const std::string& categoryId = "", const std::string& query = "");
    Result<std::vector<Category>> listCategories(const std::string& typeKey);
    Result<std::vector<ResourceVersion>> listResourceVersions(const std::string& resourceId);
    Result<std::vector<Dependency>> getResourceDependencies(const std::string& versionId);

    // Discovery (Async)
    std::future<Result<Resource>> getResourceAsync(const std::string& id);
    std::future<Result<std::vector<Resource>>> listResourcesAsync(const std::string& typeKey = "", const std::string& categoryId = "", const std::string& query = "");
    std::future<Result<std::vector<Category>>> listCategoriesAsync(const std::string& typeKey);
    std::future<Result<std::vector<ResourceVersion>>> listResourceVersionsAsync(const std::string& resourceId);
    std::future<Result<std::vector<Dependency>>> getResourceDependenciesAsync(const std::string& versionId);

    // Transfer (Sync)
    Status downloadFile(const std::string& url, const std::string& localPath, std::function<void(double)> progressCallback = nullptr);
    Status uploadFileSimple(const std::string& typeKey, const std::string& filePath, const std::string& name, std::function<void(double)> progressCallback = nullptr);
    Status uploadFileMultipart(const std::string& typeKey, const std::string& filePath, const std::string& name, std::function<void(double)> progressCallback = nullptr, int maxRetries = 3);
    
    // Transfer (Async)
    std::future<Status> downloadFileAsync(const std::string& url, const std::string& localPath, std::function<void(double)> progressCallback = nullptr);
    std::future<Status> uploadFileSimpleAsync(const std::string& typeKey, const std::string& filePath, const std::string& name, std::function<void(double)> progressCallback = nullptr);

    // High level
    Status downloadBundle(const std::string& resourceId, const std::string& targetDir);

    // Internal / Advanced
    Result<UploadTicket> requestUploadToken(const UploadTokenRequest& req);
    Status uploadFileSTS(const UploadTicket& ticket, const std::string& filePath, const std::string& endpoint = "localhost:9000");

private:
    std::unique_ptr<ClientImpl> impl_;
};

} // namespace simhub

#endif // SIMHUB_SDK_HPP

// --- Implementation ---
#ifdef SIMHUB_IMPLEMENTATION

#include "simhub/json.hpp"
#include <curl/curl.h>
#include <fstream>
#include <iostream>
#include <sys/stat.h>
#include <thread>
#include <chrono>
#include <cmath>
#include <algorithm>
#include <cstring>
#include <mutex>

#ifdef USE_AWS_SDK
#include <aws/core/Aws.h>
#include <aws/core/auth/AWSCredentials.h>
#include <aws/s3/S3Client.h>
#include <aws/s3/model/PutObjectRequest.h>
#endif

namespace simhub {

using json = nlohmann::json;

// --- Helpers ---

static size_t WriteCallback(void* contents, size_t size, size_t nmemb, void* userp) {
    ((std::string*)userp)->append((char*)contents, size * nmemb);
    return size * nmemb;
}

static size_t FileWriteCallback(void* ptr, size_t size, size_t nmemb, FILE* stream) {
    return fwrite(ptr, size, nmemb, stream);
}

struct ProgressData {
    std::function<void(double)> callback;
};

static int ProgressCallback(void* clientp, curl_off_t dltotal, curl_off_t dlnow, curl_off_t ultotal, curl_off_t ulnow) {
    auto* data = (ProgressData*)clientp;
    if (data && data->callback) {
        if (dltotal > 0) data->callback((double)dlnow / dltotal);
        else if (ultotal > 0) data->callback((double)ulnow / ultotal);
    }
    return 0;
}

// --- DTO Mapping ---

static ResourceVersion parseVersion(const json& v) {
    ResourceVersion rv;
    rv.version_num = v.value("version_num", 0);
    rv.file_size = v.value("file_size", 0LL);
    rv.download_url = v.value("download_url", "");
    rv.semver = v.value("semver", "");
    rv.state = v.value("state", "");
    if (v.contains("meta_data") && v["meta_data"].is_object()) {
        for (auto& [key, value] : v["meta_data"].items()) {
            if (value.is_string()) rv.meta_data[key] = value.get<std::string>();
            else rv.meta_data[key] = value.dump();
        }
    }
    return rv;
}

static Resource parseResource(const json& j) {
    Resource r;
    r.id = j.value("id", "");
    r.name = j.value("name", "");
    r.type_key = j.value("type_key", "");
    r.category_id = j.value("category_id", "");
    r.owner_id = j.value("owner_id", "");
    r.scope = j.value("scope", "");
    r.created_at = j.value("created_at", "");
    if (j.contains("tags") && j["tags"].is_array()) {
        r.tags = j["tags"].get<std::vector<std::string>>();
    }
    if (j.contains("latest_version") && !j["latest_version"].is_null()) {
        r.latest_version = parseVersion(j["latest_version"]);
    }
    return r;
}

// --- ClientImpl ---

class ClientImpl {
public:
    std::string baseUrl;
    std::string token;
    CURL* curl; // Keep a single CURL handle for reuse

    ClientImpl() : curl(nullptr) {
        curl = curl_easy_init();
    }

    ~ClientImpl() {
        if (curl) {
            curl_easy_cleanup(curl);
        }
    }

    void setToken(const std::string& t) { token = t; }

    Result<std::string> request(const std::string& path, const std::string& method = "GET", const std::string& body = "") {
        if (!curl) return Result<std::string>::Fail(ErrorCode::NetworkError, "Curl not initialized");

        std::string readBuffer;
        struct curl_slist* headers = NULL;
        headers = curl_slist_append(headers, "Content-Type: application/json");
        
        if (!token.empty()) {
            std::string authHeader = "Authorization: Bearer " + token;
            headers = curl_slist_append(headers, authHeader.c_str());
        }

        std::string url = baseUrl + path;

        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &readBuffer);
        
        if (method == "POST") {
            curl_easy_setopt(curl, CURLOPT_POST, 1L);
            if (!body.empty()) {
                curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body.c_str());
            }
        } else if (method == "PUT") {
            curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, "PUT");
            if (!body.empty()) {
                curl_easy_setopt(curl, CURLOPT_POSTFIELDS, body.c_str());
            }
        }

        CURLcode res = curl_easy_perform(curl);
        long http_code = 0;
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
        curl_slist_free_all(headers);

        if (res != CURLE_OK) {
            return Result<std::string>::Fail(ErrorCode::NetworkError, curl_easy_strerror(res));
        }

        if (http_code >= 400) {
            return Result<std::string>::Fail(ErrorCode::ServerError, "HTTP Error: " + std::to_string(http_code) + " - " + readBuffer);
        }

        return Result<std::string>::Success(readBuffer);
    }
};

static std::once_flag init_flag;
void Client::GlobalInit() {
    std::call_once(init_flag, []() {
        curl_global_init(CURL_GLOBAL_ALL);
#ifdef USE_AWS_SDK
        Aws::SDKOptions options;
        Aws::InitAPI(options);
#endif
    });
}

void Client::GlobalCleanup() {
    curl_global_cleanup();
#ifdef USE_AWS_SDK
    Aws::SDKOptions options;
    Aws::ShutdownAPI(options);
#endif
}

Client::Client(const std::string& b) : impl_(std::make_unique<ClientImpl>()) {
    impl_->baseUrl = b;
    if (!impl_->baseUrl.empty() && impl_->baseUrl.back() == '/') impl_->baseUrl.pop_back();
}
void Client::setToken(const std::string& token) {
    if (impl_) impl_->setToken(token);
}
const std::string& Client::getBaseUrl() const {
    return impl_->baseUrl;
}

Client::~Client() = default;

// --- Discovery Implementations ---

Result<Resource> Client::getResource(const std::string& id) {
    auto res = impl_->request("/api/v1/resources/" + id);
    if (!res.ok()) return Result<Resource>::Fail(res.code, res.message);
    try {
        return Result<Resource>::Success(parseResource(json::parse(res.value)));
    } catch (const std::exception& e) {
        return Result<Resource>::Fail(ErrorCode::ServerError, e.what());
    }
}

Result<std::vector<Resource>> Client::listResources(const std::string& t, const std::string& c, const std::string& q) {
    std::string path = "/api/v1/resources?type=" + t + "&category_id=" + c + "&query=" + q;
    auto res = impl_->request(path);
    if (!res.ok()) return Result<std::vector<Resource>>::Fail(res.code, res.message);
    
    try {
        auto j = json::parse(res.value);
        std::vector<Resource> list;
        if (j.contains("items") && j["items"].is_array()) {
            for (auto& item : j["items"]) list.push_back(parseResource(item));
        }
        return Result<std::vector<Resource>>::Success(list);
    } catch (const std::exception& e) {
        return Result<std::vector<Resource>>::Fail(ErrorCode::ServerError, e.what());
    }
}

Result<std::vector<Category>> Client::listCategories(const std::string& typeKey) {
    auto res = impl_->request("/api/v1/categories?type=" + typeKey);
    if (!res.ok()) return Result<std::vector<Category>>::Fail(res.code, res.message);
    try {
        json items = json::parse(res.value);
        std::vector<Category> list;
        if (items.is_array()) {
            for (auto& item : items) {
                Category c;
                c.id = item.value("id", "");
                c.type_key = item.value("type_key", "");
                c.name = item.value("name", "");
                c.parent_id = item.value("parent_id", "");
                list.push_back(c);
            }
        }
        return Result<std::vector<Category>>::Success(list);
    } catch (const std::exception& e) {
        return Result<std::vector<Category>>::Fail(ErrorCode::ServerError, e.what());
    }
}

Result<std::vector<ResourceVersion>> Client::listResourceVersions(const std::string& resourceId) {
    auto res = impl_->request("/api/v1/resources/" + resourceId + "/versions");
    if (!res.ok()) return Result<std::vector<ResourceVersion>>::Fail(res.code, res.message);
    try {
        json items = json::parse(res.value);
        std::vector<ResourceVersion> list;
        if (items.is_array()) {
            for (auto& item : items) list.push_back(parseVersion(item));
        }
        return Result<std::vector<ResourceVersion>>::Success(list);
    } catch (const std::exception& e) {
        return Result<std::vector<ResourceVersion>>::Fail(ErrorCode::ServerError, e.what());
    }
}

Result<std::vector<Dependency>> Client::getResourceDependencies(const std::string& versionId) {
    auto res = impl_->request("/api/v1/resources/versions/" + versionId + "/dependencies");
    if (!res.ok()) return Result<std::vector<Dependency>>::Fail(res.code, res.message);
    try {
        json items = json::parse(res.value);
        std::vector<Dependency> list;
        if (items.is_array()) {
            for (auto& item : items) {
                Dependency d;
                d.target_resource_id = item.value("target_resource_id", "");
                d.constraint = item.value("constraint", "");
                list.push_back(d);
            }
        }
        return Result<std::vector<Dependency>>::Success(list);
    } catch (const std::exception& e) {
        return Result<std::vector<Dependency>>::Fail(ErrorCode::ServerError, e.what());
    }
}

// --- Transfer Implementations ---

Status Client::downloadFile(const std::string& url, const std::string& localPath, std::function<void(double)> callback) {
    CURL* curl = curl_easy_init();
    if (!curl) return Status::Fail(ErrorCode::Unknown, "Curl init failed");

    FILE* fp = fopen(localPath.c_str(), "wb");
    if (!fp) {
        curl_easy_cleanup(curl);
        return Status::Fail(ErrorCode::FileSystemError, "Cannot open file for writing");
    }

    ProgressData pd{callback};
    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, FileWriteCallback);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, fp);
    curl_easy_setopt(curl, CURLOPT_NOPROGRESS, 0L);
    curl_easy_setopt(curl, CURLOPT_XFERINFOFUNCTION, ProgressCallback);
    curl_easy_setopt(curl, CURLOPT_XFERINFODATA, &pd);
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);

    CURLcode res = curl_easy_perform(curl);
    long http_code = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
    
    fclose(fp);
    curl_easy_cleanup(curl);

    if (res != CURLE_OK) return Status::Fail(ErrorCode::NetworkError, curl_easy_strerror(res));
    if (http_code >= 400) return Status::Fail(ErrorCode::ServerError, "Download failed with HTTP " + std::to_string(http_code));

    return Status::Success(true);
}

Result<UploadTicket> Client::requestUploadToken(const UploadTokenRequest& req) {
    json j;
    j["resource_type"] = req.resource_type;
    j["filename"] = req.filename;
    j["size"] = req.size;
    j["mode"] = req.mode.empty() ? "presigned" : req.mode;

    auto res = impl_->request("/api/v1/integration/upload/token", "POST", j.dump());
    if (!res.ok()) return Result<UploadTicket>::Fail(res.code, res.message);
    
    try {
        json r = json::parse(res.value);
        UploadTicket ticket;
        ticket.ticket_id = r.value("ticket_id", "");
        ticket.presigned_url = r.value("presigned_url", "");
        ticket.bucket = r.value("bucket", "");
        ticket.object_key = r.value("object_key", "");
        ticket.has_credentials = false;
        
        if (r.contains("credentials") && !r["credentials"].is_null()) {
            auto c = r["credentials"];
            ticket.credentials.access_key = c.value("access_key", "");
            ticket.credentials.secret_key = c.value("secret_key", "");
            ticket.credentials.session_token = c.value("session_token", "");
            ticket.credentials.expiration = c.value("expiration", "");
            ticket.has_credentials = true;
        }
        return Result<UploadTicket>::Success(ticket);
    } catch (const std::exception& e) {
        return Result<UploadTicket>::Fail(ErrorCode::ServerError, e.what());
    }
}

Status Client::uploadFileSimple(const std::string& typeKey, const std::string& filePath, const std::string& name, std::function<void(double)> callback) {
    // 1. Get Token
    struct stat st;
    if (stat(filePath.c_str(), &st) != 0) return Status::Fail(ErrorCode::FileSystemError, "File not found");
    
    UploadTokenRequest req;
    req.resource_type = typeKey;
    req.filename = filePath; // will be extracted as basename on server
    req.size = st.st_size;
    req.mode = "presigned";

    auto tokenRes = requestUploadToken(req);
    if (!tokenRes.ok()) return Status::Fail(tokenRes.code, tokenRes.message);
    
    // 2. Upload to URL
    CURL* curl = curl_easy_init();
    FILE* fp = fopen(filePath.c_str(), "rb");
    if (!fp) {
        curl_easy_cleanup(curl);
        return Status::Fail(ErrorCode::FileSystemError, "Cannot open file for reading");
    }

    ProgressData pd{callback};
    curl_easy_setopt(curl, CURLOPT_URL, tokenRes.value.presigned_url.c_str());
    curl_easy_setopt(curl, CURLOPT_UPLOAD, 1L);
    curl_easy_setopt(curl, CURLOPT_READDATA, fp);
    curl_easy_setopt(curl, CURLOPT_INFILESIZE_LARGE, (curl_off_t)st.st_size);
    curl_easy_setopt(curl, CURLOPT_NOPROGRESS, 0L);
    curl_easy_setopt(curl, CURLOPT_XFERINFOFUNCTION, ProgressCallback);
    curl_easy_setopt(curl, CURLOPT_XFERINFODATA, &pd);

    CURLcode res = curl_easy_perform(curl);
    long http_code = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
    fclose(fp);
    curl_easy_cleanup(curl);

    if (res != CURLE_OK) return Status::Fail(ErrorCode::NetworkError, curl_easy_strerror(res));
    if (http_code >= 400) return Status::Fail(ErrorCode::StorageError, "Upload failed with HTTP " + std::to_string(http_code));

    // 3. Confirm
    json j;
    j["ticket_id"] = tokenRes.value.ticket_id;
    j["type_key"] = typeKey;
    j["name"] = name;
    j["size"] = st.st_size;

    auto confRes = impl_->request("/api/v1/integration/upload/confirm", "POST", j.dump());
    if (!confRes.ok()) return Status::Fail(confRes.code, confRes.message);

    return Status::Success(true);
}

Status Client::uploadFileMultipart(const std::string& typeKey, const std::string& filePath, const std::string& name, std::function<void(double)> callback, int maxRetries) {
    // 1. 获取文件信息
    struct stat st;
    if (stat(filePath.c_str(), &st) != 0) return Status::Fail(ErrorCode::FileSystemError, "File not found");
    long long totalSize = st.st_size;

    const long long partSize = 5 * 1024 * 1024; // 5MB per part
    int partCount = (int)std::ceil((double)totalSize / partSize);

    // 2. 初始化分片上传
    json initJ;
    initJ["resource_type"] = typeKey;
    initJ["filename"] = filePath;
    initJ["part_count"] = partCount;

    auto initRes = impl_->request("/api/v1/integration/upload/multipart/init", "POST", initJ.dump());
    if (!initRes.ok()) return Status::Fail(initRes.code, initRes.message);

    std::string uploadId, objectKey;
    try {
        auto r = json::parse(initRes.value);
        uploadId = r.value("upload_id", "");
        objectKey = r.value("key", "");
    } catch (...) {
        return Status::Fail(ErrorCode::ServerError, "Failed to parse init response");
    }

    // 3. 并发上传分片
    struct PartETag {
        int part_number;
        std::string etag;
    };

    std::vector<PartETag> etags(partCount);
    std::vector<std::thread> workers;
    std::atomic<long long> uploadedBytes{0};
    std::mutex etagMutex;
    std::atomic<bool> failed{false};
    std::string errorMessage;

    const int concurrency = 4;
    std::mutex taskMutex;
    int nextPart = 1;

    auto workerFunc = [&]() {
        while (true) {
            int partNum = 0;
            {
                std::lock_guard<std::mutex> lock(taskMutex);
                if (nextPart > partCount || failed) return;
                partNum = nextPart++;
            }

            long long offset = (long long)(partNum - 1) * partSize;
            long long currentPartSize = std::min(partSize, totalSize - offset);

            // A. 获取分片 URL
            std::string partUrlQuery = "/api/v1/integration/upload/multipart/part-url?upload_id=" + uploadId + 
                                      "&key=" + objectKey + "&part_number=" + std::to_string(partNum);
            auto urlRes = impl_->request(partUrlQuery);
            if (!urlRes.ok()) {
                failed = true;
                errorMessage = urlRes.message;
                return;
            }

            std::string presignedUrl;
            try {
                presignedUrl = json::parse(urlRes.value).value("presigned_url", "");
            } catch (...) {
                failed = true;
                errorMessage = "Failed to parse part url";
                return;
            }

            // B. 上传分片至 S3
            // 为每个线程分配独立的 CURL 句柄，实现真正的并发
            CURL* pCurl = curl_easy_init();
            FILE* pFp = fopen(filePath.c_str(), "rb");
            if (!pFp) {
                failed = true;
                errorMessage = "Failed to reopen file for thread";
                curl_easy_cleanup(pCurl);
                return;
            }
            fseeko(pFp, offset, SEEK_SET);

            curl_easy_setopt(pCurl, CURLOPT_URL, presignedUrl.c_str());
            curl_easy_setopt(pCurl, CURLOPT_UPLOAD, 1L);
            curl_easy_setopt(pCurl, CURLOPT_READDATA, pFp);
            curl_easy_setopt(pCurl, CURLOPT_INFILESIZE_LARGE, (curl_off_t)currentPartSize);
            
            // 捕获 ETag
            std::string etag;
            auto headerCb = [](char* b, size_t s, size_t n, void* u) -> size_t {
                size_t len = s * n;
                std::string h(b, len);
                if (h.find("ETag:") == 0 || h.find("etag:") == 0) {
                    size_t start = h.find(":") + 1;
                    size_t end = h.find_last_not_of(" \r\n");
                    std::string val = h.substr(start, end - start + 1);
                    if (val.size() >= 2 && val.front() == '"') val = val.substr(1, val.size() - 2);
                    *((std::string*)u) = val;
                }
                return len;
            };
            curl_easy_setopt(pCurl, CURLOPT_HEADERFUNCTION, headerCb);
            curl_easy_setopt(pCurl, CURLOPT_HEADERDATA, &etag);

            CURLcode resResource = curl_easy_perform(pCurl);
            long httpCode = 0;
            curl_easy_getinfo(pCurl, CURLINFO_RESPONSE_CODE, &httpCode);
            fclose(pFp);
            curl_easy_cleanup(pCurl);

            if (resResource != CURLE_OK || httpCode >= 400) {
                failed = true;
                errorMessage = "Part " + std::to_string(partNum) + " failed: " + curl_easy_strerror(resResource);
                return;
            }

            {
                std::lock_guard<std::mutex> lock(etagMutex);
                etags[partNum - 1] = {partNum, etag};
            }

            long long nowTotal = uploadedBytes.fetch_add(currentPartSize) + currentPartSize;
            if (callback) callback((double)nowTotal / totalSize);
        }
    };

    for (int i = 0; i < concurrency; ++i) workers.emplace_back(workerFunc);
    for (auto& w : workers) w.join();

    if (failed) return Status::Fail(ErrorCode::StorageError, errorMessage);

    // 4. 完成合并
    json completeJ;
    completeJ["upload_id"] = uploadId;
    completeJ["key"] = objectKey;
    json partsJ = json::array();
    for (const auto& tag : etags) {
        partsJ.push_back({{"part_number", tag.part_number}, {"etag", tag.etag}});
    }
    completeJ["parts"] = partsJ;

    auto compRes = impl_->request("/api/v1/integration/upload/multipart/complete", "POST", completeJ.dump());
    if (!compRes.ok()) return Status::Fail(compRes.code, compRes.message);

    std::string ticketId;
    try {
        ticketId = json::parse(compRes.value).value("ticket_id", "");
    } catch (...) {
        return Status::Fail(ErrorCode::ServerError, "Failed to parse complete response");
    }

    // 5. 最终确认
    json confJ;
    confJ["ticket_id"] = ticketId;
    confJ["name"] = name;
    confJ["size"] = totalSize;
    confJ["type_key"] = typeKey;

    auto finalRes = impl_->request("/api/v1/integration/upload/confirm", "POST", confJ.dump());
    return finalRes.ok() ? Status::Success(true) : Status::Fail(finalRes.code, finalRes.message);
}

// --- Discovery (Async) ---

std::future<Result<Resource>> Client::getResourceAsync(const std::string& id) {
    return std::async(std::launch::async, [this, id]() { return getResource(id); });
}

std::future<Result<std::vector<Resource>>> Client::listResourcesAsync(const std::string& t, const std::string& c, const std::string& q) {
    return std::async(std::launch::async, [this, t, c, q]() { return listResources(t, c, q); });
}

std::future<Result<std::vector<Category>>> Client::listCategoriesAsync(const std::string& t) {
    return std::async(std::launch::async, [this, t]() { return listCategories(t); });
}

std::future<Result<std::vector<ResourceVersion>>> Client::listResourceVersionsAsync(const std::string& rid) {
    return std::async(std::launch::async, [this, rid]() { return listResourceVersions(rid); });
}

std::future<Result<std::vector<Dependency>>> Client::getResourceDependenciesAsync(const std::string& vid) {
    return std::async(std::launch::async, [this, vid]() { return getResourceDependencies(vid); });
}

// --- Transfer (Async) ---

std::future<Status> Client::downloadFileAsync(const std::string& url, const std::string& path, std::function<void(double)> cb) {
    return std::async(std::launch::async, [this, url, path, cb]() { return downloadFile(url, path, cb); });
}

std::future<Status> Client::uploadFileSimpleAsync(const std::string& tk, const std::string& path, const std::string& name, std::function<void(double)> cb) {
    return std::async(std::launch::async, [this, tk, path, name, cb]() { return uploadFileSimple(tk, path, name, cb); });
}

// --- High Level ---

Status Client::downloadBundle(const std::string& resourceId, const std::string& targetDir) {
    // Recursive Downloader Helper
    std::function<Status(const std::string&, std::set<std::string>&)> resolveAndDownload;
    
    resolveAndDownload = [&](const std::string& id, std::set<std::string>& visited) -> Status {
        if (visited.count(id)) return Status::Success(true);
        visited.insert(id);

        // 1. Get Resource Info
        auto res = getResource(id);
        if (!res.ok()) return Status::Fail(res.code, "Failed to fetch resource " + id + ": " + res.message);

        // 2. Download Latest Version
        auto latest = res.value.latest_version;
        if (!latest.download_url.empty()) {
            std::string localPath = targetDir + "/" + id + "_" + latest.semver + ".zip";
            auto dl = downloadFile(latest.download_url, localPath);
            if (!dl.ok()) return dl;
        }

        // 3. Resolve Dependencies
        auto deps = getResourceDependencies(id); // Using ID here, but ideally should use specific versionId if we had it
        if (deps.ok()) {
            for (const auto& dep : deps.value) {
                auto sub = resolveAndDownload(dep.target_resource_id, visited);
                if (!sub.ok()) return sub;
            }
        }

        return Status::Success(true);
    };

    std::set<std::string> visited;
    return resolveAndDownload(resourceId, visited);
}

Status Client::uploadFileSTS(const UploadTicket& ticket, const std::string& filePath, const std::string& endpoint) {
#ifdef USE_AWS_SDK
    Aws::Auth::AWSCredentials creds(ticket.credentials.access_key.c_str(), ticket.credentials.secret_key.c_str(), ticket.credentials.session_token.c_str());
    Aws::S3::S3ClientConfiguration clientConfig;
    clientConfig.endpointOverride = endpoint.c_str();
    clientConfig.scheme = Aws::Http::Scheme::HTTP; 

    Aws::S3::S3Client s3_client(creds, clientConfig, Aws::Client::AWSAuthV4Signer::PayloadSigningPolicy::Never, false);
    
    Aws::S3::Model::PutObjectRequest request;
    request.SetBucket(ticket.bucket.c_str());
    request.SetKey(ticket.object_key.c_str());

    auto input_data = Aws::MakeShared<Aws::FStream>("SimHubAllocation", filePath.c_str(), std::ios_base::in | std::ios_base::binary);
    request.SetBody(input_data);

    auto outcome = s3_client.PutObject(request);
    if (!outcome.IsSuccess()) {
        return Status::Fail(ErrorCode::StorageError, outcome.GetError().GetMessage().c_str());
    }
    return Status::Success(true);
#else
    return Status::Fail(ErrorCode::InvalidParam, "AWS SDK not enabled (compile with -DUSE_AWS_SDK)");
#endif
}

} // namespace simhub
#endif // SIMHUB_IMPLEMENTATION
