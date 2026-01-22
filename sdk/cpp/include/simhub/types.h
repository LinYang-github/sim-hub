#ifndef SIMHUB_SDK_TYPES_H
#define SIMHUB_SDK_TYPES_H

#include <string>
#include <vector>
#include <map>
#include <optional>

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

template <typename T>
struct Result {
    T value;
    ErrorCode code = ErrorCode::Success;
    std::string message;

    bool ok() const { return code == ErrorCode::Success; }
    
    static Result<T> Success(T val) { return {val, ErrorCode::Success, ""}; }
    static Result<T> Fail(ErrorCode c, std::string msg) { return {T(), c, msg}; }
};

// Also specify Result<bool> helper
using Status = Result<bool>;

struct STSCredentials {
    std::string access_key;
    std::string secret_key;
    std::string session_token;
    std::string expiration;
};

struct UploadTokenRequest {
    std::string resource_type;
    std::string filename;
    long long size;
    std::string checksum;
    std::string mode; // "presigned" or "sts"
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

// Multipart Upload Types
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

struct ResourceVersionDTO {
    int version_num;
    long long file_size;
    std::string download_url;
};

struct ResourceDTO {
    std::string id;
    std::string type_key;
    std::string name;
    std::string owner_id;
    std::vector<std::string> tags;
    std::string created_at;
    ResourceVersionDTO latest_version;
};

} // namespace simhub

#endif // SIMHUB_SDK_TYPES_H
