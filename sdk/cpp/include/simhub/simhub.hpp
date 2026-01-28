#ifndef SIMHUB_SDK_HPP
#define SIMHUB_SDK_HPP

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <memory>
#include <functional>
#include <mutex>

namespace simhub {

// --- Types ---

enum class ErrorCode {
    Success = 0,
    NetworkError,
    ServerError,
    InvalidParam,
    FileSystemError,
    StorageError,
    Unknown
};

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

struct ResourceVersionDTO {
    int version_num;
    long long file_size;
    std::string download_url;
    std::string semver;
    std::string state;
    std::map<std::string, std::string> meta_data;
};

struct ResourceDTO {
    std::string id;
    std::string type_key;
    std::string name;
    std::string owner_id;
    std::vector<std::string> tags;
    std::string created_at;
    ResourceVersionDTO latest_version;
    std::string scope; 
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

struct ConfirmUploadRequest {
    std::string ticket_id;
    std::string type_key;
    std::string name;
    std::string owner_id;
    long long size;
    std::map<std::string, std::string> extra_meta;
};

struct MultipartInitRequest {
    std::string resource_type;
    std::string filename;
};

struct MultipartInitResponse {
    std::string ticket_id;
    std::string upload_id;
    std::string bucket;
    std::string object_key;
};

struct PartInfo {
    int part_number;
    std::string etag;
};

struct MultipartCompleteRequest {
    std::string ticket_id;
    std::string upload_id;
    std::vector<PartInfo> parts;
    std::string type_key;
    std::string name;
    std::string owner_id;
    std::map<std::string, std::string> extra_meta;
};


// --- Client Interface ---
class ClientImpl;

/**
 * SimHub SDK Client
 * A thread-safe, easy-to-use C++ client for SimHub.
 */
class Client {
public:
    /**
     * @param baseUrl Backend Base URL (e.g. http://localhost:30030)
     */
    explicit Client(const std::string& baseUrl);
    ~Client();

    // Disallow copy
    Client(const Client&) = delete;
    Client& operator=(const Client&) = delete;

    /**
     * Initialize the SDK. Must be called once at application startup.
     */
    static void GlobalInit();

    /**
     * Cleanup the SDK. Must be called once before application exit.
     */
    static void GlobalCleanup();

    // --- Discovery ---
    /**
     * Get details of a specific resource by ID
     */
    Result<ResourceDTO> getResource(const std::string& id);

    /**
     * List resources with optional filters
     */
    Result<std::vector<ResourceDTO>> listResources(const std::string& typeKey = "", const std::string& categoryId = "");

    // --- Transfer ---
    /**
     * Download a file from a URL to a local path
     */
    Status downloadFile(const std::string& url, const std::string& localPath, std::function<void(double)> progressCallback = nullptr);

    /**
     * Upload a file (Simple mode, < 5GB)
     */
    Status uploadFileSimple(const std::string& typeKey, const std::string& filePath, const std::string& name, std::function<void(double)> progressCallback = nullptr);

    /**
     * Upload a large file (Multipart mode, recommended for > 100MB)
     */
    Status uploadFileMultipart(const std::string& typeKey, const std::string& filePath, const std::string& name, std::function<void(double)> progressCallback = nullptr, int maxRetries = 3);
    
    // --- Advanced / Internal ---
    Result<UploadTicket> requestUploadToken(const UploadTokenRequest& req);
    Status confirmUpload(const ConfirmUploadRequest& req);
    Status uploadFileToUrl(const std::string& url, const std::string& filePath, std::function<void(double)> progressCallback);
    Result<MultipartInitResponse> initMultipartUpload(const MultipartInitRequest& req);
    Result<std::string> getMultipartPartURL(const std::string& ticketId, const std::string& uploadId, int partNumber);
    Status completeMultipartUpload(const MultipartCompleteRequest& req);

    /**
     * Upload file using AWS SDK (STS Credential Mode).
     * Requires the SDK to be compiled with -DUSE_AWS_SDK and linked against aws-cpp-sdk-s3.
     */
    Status uploadFileSTS(const UploadTicket& ticket, const std::string& filePath, const std::string& endpoint = "localhost:9000");

private:
    std::unique_ptr<ClientImpl> impl_;
};

} // namespace simhub

#endif // SIMHUB_SDK_HPP
