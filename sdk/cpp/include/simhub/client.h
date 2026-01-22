#ifndef SIMHUB_SDK_CLIENT_H
#define SIMHUB_SDK_CLIENT_H

#include "types.h"
#include <memory>
#include <functional>

namespace simhub {

class ClientImpl;

/**
 * SimHub SDK 客户端
 */
class Client {
public:
    /**
     * @param baseUrl 后端基础 URL (例如 http://localhost:30030)
     */
    explicit Client(const std::string& baseUrl);
    ~Client();

    // 禁止拷贝
    Client(const Client&) = delete;
    Client& operator=(const Client&) = delete;

    /**
     * 申请上传令牌 (Token/Ticket)
     */
    UploadTicket requestUploadToken(const UploadTokenRequest& req);

    /**
     * 确认上传完成
     */
    bool confirmUpload(const ConfirmUploadRequest& req);

    /**
     * 获取资源详情
     */
    ResourceDTO getResource(const std::string& id);

    /**
     * 基础 HTTP 上传 (PUT 方法，适用于 Presigned URL)
     * @param url 预签名 URL
     * @param filePath 本地文件路径
     * @param progressCallback 进度回调 (0.0 - 1.0)
     */
    bool uploadFileSimple(const std::string& url, 
                         const std::string& filePath,
                         std::function<void(double)> progressCallback = nullptr);

    /**
     * 使用 STS 凭证进行 S3 原生上传 (需要 AWS SDK)
     * 注意：此方法仅在链接了 AWS SDK 的情况下可用，否则将抛出异常或返回失败。
     */
    bool uploadFileSTS(const UploadTicket& ticket, 
                      const std::string& filePath,
                      const std::string& endpoint = "localhost:9000");

private:
    std::unique_ptr<ClientImpl> impl_;
};

} // namespace simhub

#endif // SIMHUB_SDK_CLIENT_H
