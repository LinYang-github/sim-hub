#include "simhub/simhub.hpp"
#include "simhub/json.hpp"
#include <curl/curl.h>
#include <fstream>
#include <iostream>
#include <sys/stat.h>
#include <mutex>
#include <thread>
#include <chrono>
#include <cmath>
#include <memory>
#include <algorithm>
#include <cstring>

#ifdef USE_AWS_SDK
#include <aws/core/Aws.h>
#include <aws/core/auth/AWSCredentials.h>
#include <aws/s3/S3Client.h>
#include <aws/s3/model/PutObjectRequest.h>
#endif

using json = nlohmann::json;

namespace simhub {

// --- Implementation Details ---

class ClientImpl {
public:
    std::string baseUrl;

    static size_t WriteCallback(void* contents, size_t size, size_t nmemb, void* userp) {
        ((std::string*)userp)->append((char*)contents, size * nmemb);
        return size * nmemb;
    }

    struct HttpResponse {
        long code;
        std::string body;
        std::string error;
        ErrorCode errorCode;
    };

    HttpResponse post(const std::string& endpoint, const json& body) {
        return request("POST", endpoint, body.dump());
    }

    HttpResponse get(const std::string& endpoint) {
        return request("GET", endpoint, "");
    }

    HttpResponse request(const std::string& method, const std::string& endpoint, const std::string& bodyData) {
        CURL* curl = curl_easy_init();
        if(!curl) return {0, "", "Failed to init curl", ErrorCode::Unknown};

        std::string url = baseUrl + endpoint;
        std::string readBuffer;

        struct curl_slist* headers = NULL;
        headers = curl_slist_append(headers, "Content-Type: application/json");

        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &readBuffer);

        if (method == "POST") {
            curl_easy_setopt(curl, CURLOPT_POST, 1L);
            curl_easy_setopt(curl, CURLOPT_POSTFIELDS, bodyData.c_str());
        }

        CURLcode res = curl_easy_perform(curl);
        long http_code = 0;
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);

        curl_easy_cleanup(curl);
        curl_slist_free_all(headers);

        if(res != CURLE_OK) {
            return {0, "", std::string("Network error: ") + curl_easy_strerror(res), ErrorCode::NetworkError};
        }

        return {http_code, readBuffer, "", ErrorCode::Success};
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

Client::Client(const std::string& baseUrl) : impl_(std::make_unique<ClientImpl>()) {
    impl_->baseUrl = baseUrl;
}

Client::~Client() = default;

// --- DTO Parsing Helpers ---

static ResourceDTO parseResourceDTO(const json& j) {
    ResourceDTO dto;
    dto.id = j.value("id", "");
    dto.name = j.value("name", "");
    dto.type_key = j.value("type_key", "");
    dto.owner_id = j.value("owner_id", "");
    dto.scope = j.value("scope", "");
    if (j.contains("tags") && j["tags"].is_array()) {
        for (auto& t : j["tags"]) {
            if (t.is_string()) dto.tags.push_back(t.get<std::string>());
        }
    }
    dto.created_at = j.value("created_at", "");
    if (j.contains("latest_version") && !j["latest_version"].is_null()) {
        auto v = j["latest_version"];
        dto.latest_version.version_num = v.value("version_num", 0);
        dto.latest_version.file_size = v.value("file_size", 0LL);
        dto.latest_version.download_url = v.value("download_url", "");
        dto.latest_version.semver = v.value("semver", "");
        dto.latest_version.state = v.value("state", "");
        if (v.contains("meta_data") && v["meta_data"].is_object()) {
            for (auto& [k, val] : v["meta_data"].items()) {
               if(val.is_string()) dto.latest_version.meta_data[k] = val.get<std::string>();
               else dto.latest_version.meta_data[k] = val.dump();
            }
        }
    }
    return dto;
}

// --- Public APIs ---

Result<ResourceDTO> Client::getResource(const std::string& id) {
    auto res = impl_->get("/api/v1/resources/" + id);
    if (res.errorCode != ErrorCode::Success) return Result<ResourceDTO>::Fail(res.errorCode, res.error);
    if (res.code >= 400) return Result<ResourceDTO>::Fail(ErrorCode::ServerError, "HTTP " + std::to_string(res.code) + ": " + res.body);
    
    try {
        return Result<ResourceDTO>::Success(parseResourceDTO(json::parse(res.body)));
    } catch (...) {
        return Result<ResourceDTO>::Fail(ErrorCode::ServerError, "JSON Parse Error");
    }
}

Result<std::vector<ResourceDTO>> Client::listResources(const std::string& typeKey, const std::string& categoryId) {
    std::string endpoint = "/api/v1/resources?";
    if (!typeKey.empty()) endpoint += "type=" + typeKey + "&";
    if (!categoryId.empty()) endpoint += "category_id=" + categoryId + "&";

    auto res = impl_->get(endpoint);
    if (res.errorCode != ErrorCode::Success) return Result<std::vector<ResourceDTO>>::Fail(res.errorCode, res.error);
    if (res.code >= 400) return Result<std::vector<ResourceDTO>>::Fail(ErrorCode::ServerError, "HTTP " + std::to_string(res.code));

    try {
        json j = json::parse(res.body);
        std::vector<ResourceDTO> list;
        if (j.contains("items") && j["items"].is_array()) {
            for (auto& item : j["items"]) {
                list.push_back(parseResourceDTO(item));
            }
        }
        return Result<std::vector<ResourceDTO>>::Success(list);
    } catch (...) {
        return Result<std::vector<ResourceDTO>>::Fail(ErrorCode::ServerError, "JSON Parse Error");
    }
}

Result<UploadTicket> Client::requestUploadToken(const UploadTokenRequest& req) {
    json body = {
        {"resource_type", req.resource_type},
        {"filename", req.filename},
        {"size", req.size},
        {"checksum", req.checksum},
        {"mode", req.mode}
    };

    auto res = impl_->post("/api/v1/integration/upload/token", body);
    if (res.errorCode != ErrorCode::Success) return Result<UploadTicket>::Fail(res.errorCode, res.error);
    if (res.code >= 400) return Result<UploadTicket>::Fail(ErrorCode::ServerError, res.body);

    try {
        json j = json::parse(res.body);
        if (j.contains("error")) return Result<UploadTicket>::Fail(ErrorCode::ServerError, j["error"]);

        UploadTicket ticket;
        ticket.ticket_id = j.value("ticket_id", "");
        ticket.presigned_url = j.value("presigned_url", "");
        ticket.bucket = j.value("bucket", "");
        ticket.object_key = j.value("object_key", "");
        if (j.contains("credentials") && !j["credentials"].is_null()) {
             auto c = j["credentials"];
             ticket.credentials.access_key = c.value("access_key", "");
             ticket.credentials.secret_key = c.value("secret_key", "");
             ticket.credentials.session_token = c.value("session_token", "");
             ticket.credentials.expiration = c.value("expiration", "");
             ticket.has_credentials = true;
        } else {
             ticket.has_credentials = false;
        }
        return Result<UploadTicket>::Success(ticket);
    } catch (...) {
        return Result<UploadTicket>::Fail(ErrorCode::ServerError, "JSON Parse Error");
    }
}

Status Client::confirmUpload(const ConfirmUploadRequest& req) {
    json body = {
        {"ticket_id", req.ticket_id},
        {"type_key", req.type_key},
        {"name", req.name},
        {"owner_id", req.owner_id},
        {"size", req.size},
        {"extra_meta", req.extra_meta}
    };
    auto res = impl_->post("/api/v1/integration/upload/confirm", body);
    if (res.errorCode != ErrorCode::Success) return Status::Fail(res.errorCode, res.error);
    if (res.code >= 400) return Status::Fail(ErrorCode::ServerError, "HTTP " + std::to_string(res.code));
    return Status::Success(true);
}

// Download Implementation
static size_t DownloadWriteCallback(void* ptr, size_t size, size_t nmemb, FILE* stream) {
    return fwrite(ptr, size, nmemb, stream);
}

static size_t ProgressCallbackProxy(void* clientp, curl_off_t dltotal, curl_off_t dlnow, curl_off_t ultotal, curl_off_t ulnow) {
    auto* cb = static_cast<std::function<void(double)>*>(clientp);
    if (cb && *cb) {
        double dlt = static_cast<double>(dltotal);
        double ult = static_cast<double>(ultotal);
        if (ult > 0) (*cb)(static_cast<double>(ulnow) / ult);
        else if (dlt > 0) (*cb)(static_cast<double>(dlnow) / dlt);
    }
    return 0;
}

Status Client::downloadFile(const std::string& url, const std::string& localPath, std::function<void(double)> progressCallback) {
    CURL* curl = curl_easy_init();
    if (!curl) return Status::Fail(ErrorCode::Unknown, "CURL init failed");

    FILE* fp = fopen(localPath.c_str(), "wb");
    if (!fp) {
        curl_easy_cleanup(curl);
        return Status::Fail(ErrorCode::FileSystemError, "Failed to open file for writing: " + localPath);
    }

    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, DownloadWriteCallback);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, fp);
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);

    if (progressCallback) {
        curl_easy_setopt(curl, CURLOPT_XFERINFOFUNCTION, ProgressCallbackProxy);
        curl_easy_setopt(curl, CURLOPT_XFERINFODATA, &progressCallback);
        curl_easy_setopt(curl, CURLOPT_NOPROGRESS, 0L);
    }

    auto res = curl_easy_perform(curl);
    long http_code = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
    fclose(fp);
    curl_easy_cleanup(curl);

    if (res != CURLE_OK) return Status::Fail(ErrorCode::NetworkError, curl_easy_strerror(res));
    if (http_code >= 400) return Status::Fail(ErrorCode::StorageError, "HTTP " + std::to_string(http_code));
    return Status::Success(true);
}

Status Client::uploadFileToUrl(const std::string& url, const std::string& filePath, std::function<void(double)> progressCallback) {
    CURL* curl = curl_easy_init();
    if(!curl) return Status::Fail(ErrorCode::Unknown, "CURL init failed");

    FILE* fd = fopen(filePath.c_str(), "rb");
    if(!fd) {
        curl_easy_cleanup(curl);
        return Status::Fail(ErrorCode::FileSystemError, "Failed to open file: " + filePath);
    }

    struct stat file_info;
    if (fstat(fileno(fd), &file_info) != 0) {
        fclose(fd);
        curl_easy_cleanup(curl);
        return Status::Fail(ErrorCode::FileSystemError, "Failed to stat file");
    }

    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl, CURLOPT_UPLOAD, 1L);
    curl_easy_setopt(curl, CURLOPT_READDATA, fd);
    curl_easy_setopt(curl, CURLOPT_INFILESIZE_LARGE, (curl_off_t)file_info.st_size);

    if (progressCallback) {
        curl_easy_setopt(curl, CURLOPT_XFERINFOFUNCTION, ProgressCallbackProxy);
        curl_easy_setopt(curl, CURLOPT_XFERINFODATA, &progressCallback);
        curl_easy_setopt(curl, CURLOPT_NOPROGRESS, 0L);
    }

    auto res = curl_easy_perform(curl);
    long http_code = 0;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);

    fclose(fd);
    curl_easy_cleanup(curl);

    if (res != CURLE_OK) return Status::Fail(ErrorCode::NetworkError, curl_easy_strerror(res));
    if (http_code >= 400) return Status::Fail(ErrorCode::StorageError, "HTTP " + std::to_string(http_code));
    return Status::Success(true);
}

Status Client::uploadFileSimple(const std::string& typeKey, const std::string& filePath, const std::string& name, std::function<void(double)> progressCallback) {
    // 1. Get File Size
    struct stat file_info;
    if (stat(filePath.c_str(), &file_info) != 0) {
        return Status::Fail(ErrorCode::FileSystemError, "Failed to stat file");
    }

    // 2. Request Token
    std::string filename = filePath.substr(filePath.find_last_of("/\\") + 1);
    UploadTokenRequest req = {typeKey, filename, (long long)file_info.st_size, "", "presigned"};
    auto tokenRes = requestUploadToken(req);
    if (!tokenRes.ok()) return Status::Fail(tokenRes.code, "Failed to get upload token: " + tokenRes.message);

    // 3. Upload to Presigned URL
    auto uploadStatus = uploadFileToUrl(tokenRes.value.presigned_url, filePath, progressCallback);
    if (!uploadStatus.ok()) return uploadStatus;

    // 4. Confirm
    ConfirmUploadRequest confirmReq = {
        tokenRes.value.ticket_id,
        typeKey,
        name,
        "cpp_sdk_user", 
        (long long)file_info.st_size,
        {{"uploaded_by", "cpp_sdk"}}
    };
    return confirmUpload(confirmReq);
}

// Memory Reader for multipart chunks
struct MemoryReader {
    const char* data;
    size_t size;
    size_t pos;
};

static size_t ReadMemoryCallback(char* dest, size_t size, size_t nmemb, void* userp) {
    MemoryReader* reader = static_cast<MemoryReader*>(userp);
    size_t buffer_size = size * nmemb;
    size_t left = reader->size - reader->pos;
    size_t to_copy = std::min(buffer_size, left);
    if (to_copy > 0) {
        memcpy(dest, reader->data + reader->pos, to_copy);
        reader->pos += to_copy;
        return to_copy;
    }
    return 0;
}

static size_t HeaderCallback(char* buffer, size_t size, size_t nitems, void* userdata) {
    std::string* headers = static_cast<std::string*>(userdata);
    headers->append(buffer, size * nitems);
    return size * nitems;
}

Result<MultipartInitResponse> Client::initMultipartUpload(const MultipartInitRequest& req) {
    json body = {{"resource_type", req.resource_type}, {"filename", req.filename}};
    auto res = impl_->post("/api/v1/integration/upload/multipart/init", body);
    if (res.errorCode != ErrorCode::Success) return Result<MultipartInitResponse>::Fail(res.errorCode, res.error);
    if (res.code >= 400) return Result<MultipartInitResponse>::Fail(ErrorCode::ServerError, res.body);

    try {
        json j = json::parse(res.body);
        MultipartInitResponse resp;
        resp.ticket_id = j.value("ticket_id", "");
        resp.upload_id = j.value("upload_id", "");
        resp.bucket = j.value("bucket", "");
        resp.object_key = j.value("object_key", "");
        return Result<MultipartInitResponse>::Success(resp);
    } catch (...) {
        return Result<MultipartInitResponse>::Fail(ErrorCode::ServerError, "JSON Parse Error");
    }
}

Result<std::string> Client::getMultipartPartURL(const std::string& ticketId, const std::string& uploadId, int partNumber) {
    json body = {{"ticket_id", ticketId}, {"upload_id", uploadId}, {"part_number", partNumber}};
    auto res = impl_->post("/api/v1/integration/upload/multipart/part-url", body);
     if (res.errorCode != ErrorCode::Success) return Result<std::string>::Fail(res.errorCode, res.error);
     if (res.code >= 400) return Result<std::string>::Fail(ErrorCode::ServerError, res.body);
     try {
         return Result<std::string>::Success(json::parse(res.body).value("url", ""));
     } catch(...) { return Result<std::string>::Fail(ErrorCode::ServerError, "JSON Parse Error"); }
}

Status Client::completeMultipartUpload(const MultipartCompleteRequest& req) {
    json parts = json::array();
    for (const auto& p : req.parts) parts.push_back({{"part_number", p.part_number}, {"etag", p.etag}});
    
    json body = {
        {"ticket_id", req.ticket_id},
        {"upload_id", req.upload_id},
        {"parts", parts},
        {"type_key", req.type_key},
        {"name", req.name},
        {"owner_id", req.owner_id},
        {"extra_meta", req.extra_meta}
    };
    
    auto res = impl_->post("/api/v1/integration/upload/multipart/complete", body);
    if (res.errorCode != ErrorCode::Success) return Status::Fail(res.errorCode, res.error);
    if (res.code >= 400) return Status::Fail(ErrorCode::ServerError, "HTTP " + std::to_string(res.code));
    return Status::Success(true);
}

Status Client::uploadFileMultipart(const std::string& typeKey, const std::string& filePath, const std::string& name, std::function<void(double)> progressCallback, int maxRetries) {
    std::string filename = filePath.substr(filePath.find_last_of("/\\") + 1);
    auto initRes = initMultipartUpload({typeKey, filename});
    if (!initRes.ok()) return Status::Fail(initRes.code, "Init failed: " + initRes.message);
    auto& initResp = initRes.value;

    std::ifstream file(filePath, std::ios::binary | std::ios::ate);
    if (!file.is_open()) return Status::Fail(ErrorCode::FileSystemError, "Cannot open file: " + filePath);
    std::streamsize fileSize = file.tellg();
    file.seekg(0, std::ios::beg);

    const size_t CHUNK_SIZE = 5 * 1024 * 1024; // 5MB
    std::vector<PartInfo> completedParts;
    int totalParts = (fileSize + CHUNK_SIZE - 1) / CHUNK_SIZE;

    std::vector<char> buffer(CHUNK_SIZE);
    for (int i = 1; i <= totalParts; ++i) {
        std::streamsize toRead = std::min((std::streamsize)CHUNK_SIZE, fileSize - (std::streamsize)file.tellg());
        file.read(buffer.data(), toRead);

        std::string etag;
        bool partSuccess = false;
        std::string lastError;

        for (int retry = 0; retry <= maxRetries; ++retry) {
            if (retry > 0) std::this_thread::sleep_for(std::chrono::milliseconds((int)std::pow(2, retry - 1) * 1000));

            auto urlRes = getMultipartPartURL(initResp.ticket_id, initResp.upload_id, i);
            if (!urlRes.ok()) { lastError = urlRes.message; continue; }

            CURL* curl = curl_easy_init();
            std::string headerBuffer;
            curl_easy_setopt(curl, CURLOPT_URL, urlRes.value.c_str());
            curl_easy_setopt(curl, CURLOPT_UPLOAD, 1L);
            MemoryReader reader = { buffer.data(), (size_t)toRead, 0 };
            curl_easy_setopt(curl, CURLOPT_READFUNCTION, ReadMemoryCallback);
            curl_easy_setopt(curl, CURLOPT_READDATA, &reader);
            curl_easy_setopt(curl, CURLOPT_INFILESIZE_LARGE, (curl_off_t)toRead);
            curl_easy_setopt(curl, CURLOPT_HEADERFUNCTION, HeaderCallback);
            curl_easy_setopt(curl, CURLOPT_HEADERDATA, &headerBuffer);

            struct curl_slist* headers = NULL;
            //headers = curl_slist_append(headers, "Content-Type:"); // Not needed for presigned typically unless signed
            headers = curl_slist_append(headers, "Expect:");
            curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
            
            auto res = curl_easy_perform(curl);
            long http_code = 0;
            curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
            curl_slist_free_all(headers);
            curl_easy_cleanup(curl);

            if (res == CURLE_OK && http_code < 400) {
                 size_t etagPos = headerBuffer.find("ETag: ");
                 if (etagPos != std::string::npos) {
                     size_t start = etagPos + 6;
                     size_t end = headerBuffer.find("\r\n", start);
                     etag = headerBuffer.substr(start, end - start);
                     if (etag.size() >= 2 && etag.front() == '"' && etag.back() == '"') etag = etag.substr(1, etag.size() - 2);
                     partSuccess = true;
                     break;
                 } else { lastError = "ETag missing"; }
            } else { lastError = "HTTP " + std::to_string(http_code); }
        }

        if (!partSuccess) return Status::Fail(ErrorCode::NetworkError, "Failed part " + std::to_string(i) + ": " + lastError);
        completedParts.push_back({i, etag});
        if (progressCallback) progressCallback((double)i / totalParts);
    }

    MultipartCompleteRequest completeReq;
    completeReq.ticket_id = initResp.ticket_id;
    completeReq.upload_id = initResp.upload_id;
    completeReq.parts = completedParts;
    completeReq.type_key = typeKey;
    completeReq.name = name;
    completeReq.owner_id = "cpp_sdk_multipart";
    return completeMultipartUpload(completeReq);
}

Status Client::uploadFileSTS(const UploadTicket& ticket, const std::string& filePath, const std::string& endpoint) {
#ifdef USE_AWS_SDK
    if (!ticket.has_credentials) return Status::Fail(ErrorCode::InvalidParam, "Ticket has no STS credentials");

    // Initialize AWS SDK (usually done globally, but we assume GlobalInit handles basic API Init if we were extending it,
    // or the user handles it. For now, let's assume the user of this method has initialized AWS SDK or we do it lazily/globally inside GlobalInit)
    // Note: In a real "single file" SDK, we might want to hide this complexity, but mixing libcurl and aws-sdk requires care.
    // Here we use the raw AWS S3 Client with provided credentials.

    Aws::Auth::AWSCredentials awsCreds(
        ticket.credentials.access_key.c_str(), 
        ticket.credentials.secret_key.c_str(), 
        ticket.credentials.session_token.c_str()
    );

    Aws::Client::ClientConfiguration clientConfig;
    clientConfig.endpointOverride = endpoint.c_str();
    clientConfig.scheme = Aws::Http::Scheme::HTTP;
    clientConfig.verifySSL = false; // For local MinIO/dev

    Aws::S3::S3Client s3_client(awsCreds, clientConfig, 
        Aws::Client::AWSAuthV4Signer::PayloadSigningPolicy::Never, false);

    Aws::S3::Model::PutObjectRequest request;
    request.SetBucket(ticket.bucket.c_str());
    request.SetKey(ticket.object_key.c_str());

    // Fix: Aws::MakeShared requires a tag name as first argument
    auto inputData = Aws::MakeShared<Aws::FStream>("SimHubSDK",
        filePath.c_str(), 
        std::ios_base::in | std::ios_base::binary);

    if (!inputData->is_open()) return Status::Fail(ErrorCode::FileSystemError, "AWS SDK failed to open file");

    request.SetBody(inputData);

    auto outcome = s3_client.PutObject(request);
    if (!outcome.IsSuccess()) {
        return Status::Fail(ErrorCode::StorageError, outcome.GetError().GetMessage().c_str());
    }
    return Status::Success(true);
#else
    return Status::Fail(ErrorCode::Unknown, "SDK not built with AWS support (define USE_AWS_SDK)");
#endif
}

} // namespace simhub
