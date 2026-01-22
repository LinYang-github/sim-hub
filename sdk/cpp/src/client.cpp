#include "simhub/client.h"
#include "simhub/json.hpp"
#include <curl/curl.h>
#include <fstream>
#include <iostream>
#include <sys/stat.h>

#ifdef USE_AWS_SDK
#include <aws/core/Aws.h>
#include <aws/core/auth/AWSCredentials.h>
#include <aws/s3/S3Client.h>
#include <aws/s3/model/PutObjectRequest.h>
#endif

using json = nlohmann::json;

namespace simhub {

// 内部实现类
class ClientImpl {
public:
    std::string baseUrl;

    static size_t WriteCallback(void* contents, size_t size, size_t nmemb, void* userp) {
        ((std::string*)userp)->append((char*)contents, size * nmemb);
        return size * nmemb;
    }

    json post(const std::string& endpoint, const json& body) {
        CURL* curl;
        CURLcode res;
        std::string readBuffer;

        curl = curl_easy_init();
        if(!curl) return {{"error", "Failed to init curl"}};

        std::string url = baseUrl + endpoint;
        std::string jsonStr = body.dump();

        struct curl_slist* headers = NULL;
        headers = curl_slist_append(headers, "Content-Type: application/json");

        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, jsonStr.c_str());
        curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &readBuffer);

        res = curl_easy_perform(curl);
        
        long http_code = 0;
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);

        curl_easy_cleanup(curl);
        curl_slist_free_all(headers);

        if(res != CURLE_OK) {
            return {{"error", std::string("Network error: ") + curl_easy_strerror(res)}};
        }

        try {
            return json::parse(readBuffer);
        } catch (...) {
            return {{"error", "Failed to parse JSON response"}};
        }
    }
};

Client::Client(const std::string& baseUrl) : impl_(std::make_unique<ClientImpl>()) {
    impl_->baseUrl = baseUrl;
    curl_global_init(CURL_GLOBAL_ALL);
}

Client::~Client() {
    curl_global_cleanup();
}

UploadTicket Client::requestUploadToken(const UploadTokenRequest& req) {
    json body = {
        {"resource_type", req.resource_type},
        {"filename", req.filename},
        {"size", req.size},
        {"checksum", req.checksum},
        {"mode", req.mode}
    };

    auto res = impl_->post("/api/v1/integration/upload/token", body);
    
    UploadTicket ticket;
    if (res.contains("error")) {
        ticket.ticket_id = ""; // Error indicator
        return ticket;
    }

    ticket.ticket_id = res.value("ticket_id", "");
    ticket.presigned_url = res.value("presigned_url", "");
    ticket.bucket = res.value("bucket", "");
    ticket.object_key = res.value("object_key", "");
    
    if (res.contains("credentials") && !res["credentials"].is_null()) {
        auto c = res["credentials"];
        ticket.credentials.access_key = c.value("access_key", "");
        ticket.credentials.secret_key = c.value("secret_key", "");
        ticket.credentials.session_token = c.value("session_token", "");
        ticket.credentials.expiration = c.value("expiration", "");
        ticket.has_credentials = true;
    } else {
        ticket.has_credentials = false;
    }

    return ticket;
}

bool Client::confirmUpload(const ConfirmUploadRequest& req) {
    json body = {
        {"ticket_id", req.ticket_id},
        {"type_key", req.type_key},
        {"name", req.name},
        {"owner_id", req.owner_id},
        {"size", req.size},
        {"extra_meta", req.extra_meta}
    };

    auto res = impl_->post("/api/v1/integration/upload/confirm", body);
    return res.contains("code") && (res["code"] == 200 || res["code"] == 201);
}

ResourceDTO Client::getResource(const std::string& id) {
    CURL* curl = curl_easy_init();
    std::string readBuffer;
    ResourceDTO dto;

    if(curl) {
        std::string url = impl_->baseUrl + "/api/v1/resources/" + id;
        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, ClientImpl::WriteCallback);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &readBuffer);
        
        auto res = curl_easy_perform(curl);
        curl_easy_cleanup(curl);

        if(res == CURLE_OK) {
            auto j = json::parse(readBuffer);
            dto.id = j.value("id", "");
            dto.name = j.value("name", "");
            dto.type_key = j.value("type_key", "");
            if (j.contains("latest_version")) {
                auto v = j["latest_version"];
                dto.latest_version.version_num = v.value("version_num", 0);
                dto.latest_version.download_url = v.value("download_url", "");
            }
        }
    }
    return dto;
}

static size_t ProgressCallbackProxy(void* clientp, curl_off_t dltotal, curl_off_t dlnow, curl_off_t ultotal, curl_off_t ulnow) {
    auto* cb = static_cast<std::function<void(double)>*>(clientp);
    if (cb && *cb && ultotal > 0) {
        (*cb)(static_cast<double>(ulnow) / static_cast<double>(ultotal));
    }
    return 0;
}

bool Client::uploadFileSimple(const std::string& url, const std::string& filePath, std::function<void(double)> progressCallback) {
    CURL* curl = curl_easy_init();
    if(!curl) return false;

    FILE* fd = fopen(filePath.c_str(), "rb");
    if(!fd) return false;

    struct stat file_info;
    fstat(fileno(fd), &file_info);

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
    
    fclose(fd);
    curl_easy_cleanup(curl);
    return res == CURLE_OK;
}

bool Client::uploadFileSTS(const UploadTicket& ticket, const std::string& filePath, const std::string& endpoint) {
#ifdef USE_AWS_SDK
    if (!ticket.has_credentials) return false;

    Aws::Auth::AWSCredentials awsCreds(
        ticket.credentials.access_key.c_str(), 
        ticket.credentials.secret_key.c_str(), 
        ticket.credentials.session_token.c_str()
    );

    Aws::Client::ClientConfiguration clientConfig;
    clientConfig.endpointOverride = endpoint.c_str();
    clientConfig.scheme = Aws::Http::Scheme::HTTP;
    clientConfig.verifySSL = false;

    Aws::S3::S3Client s3_client(awsCreds, clientConfig, 
        Aws::Client::AWSAuthV4Signer::PayloadSigningPolicy::Never, false);

    Aws::S3::Model::PutObjectRequest request;
    request.SetBucket(ticket.bucket.c_str());
    request.SetKey(ticket.object_key.c_str());

    auto inputData = Aws::MakeShared<Aws::FStream>("SimHubSDK",
        filePath.c_str(), 
        std::ios_base::in | std::ios_base::binary);

    request.SetBody(inputData);

    auto outcome = s3_client.PutObject(request);
    return outcome.IsSuccess();
#else
    std::cerr << "SDK Error: uploadFileSTS requires AWS SDK. Rebuild with -DUSE_AWS_SDK=ON" << std::endl;
    return false;
#endif
}

} // namespace simhub
