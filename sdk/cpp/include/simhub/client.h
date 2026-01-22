#ifndef SIMHUB_SDK_CLIENT_H
#define SIMHUB_SDK_CLIENT_H

#include "types.h"
#include <memory>
#include <functional>
#include <mutex>

namespace simhub {

class ClientImpl;

/**
 * SimHub SDK 客户端
 */
class Client {
public:
    /**
     * 全局初始化 SDK (线程安全)
     * 必须在程序启动时调用一次
     */
    static void GlobalInit();

    /**
     * 全局清理 SDK
     * 必须在程序退出前调用一次
     */
    static void GlobalCleanup();

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
    Result<UploadTicket> requestUploadToken(const UploadTokenRequest& req);

    /**
     * 确认上传完成
     */
    Status confirmUpload(const ConfirmUploadRequest& req);

    /**
     * 获取资源详情
     */
    Result<ResourceDTO> getResource(const std::string& id);

    /**
     * 基础 HTTP 上传 (PUT 方法，适用于 Presigned URL)
     * @param url 预签名 URL
     * @param filePath 本地文件路径
     * @param progressCallback 进度回调 (0.0 - 1.0)
     */
    Status uploadFileSimple(const std::string& url, 
                          const std::string& filePath,
                          std::function<void(double)> progressCallback = nullptr);

    /**
     * 初始化分片上传
     */
    Result<MultipartInitResponse> initMultipartUpload(const MultipartInitRequest& req);

    /**
     * 获取分片上传预签名 URL
     */
    Result<std::string> getMultipartPartURL(const std::string& ticketId, const std::string& uploadId, int partNumber);

    /**
     * 完成分片上传
     */
    Status completeMultipartUpload(const MultipartCompleteRequest& req);

    /**
     * 直接执行大文件分片上传 (自动切片、并发上传、合并确认)
     * @param typeKey 资源类型 (如 map_terrain)
     * @param filePath 本地文件路径
     * @param name 资源名称
     * @param progressCallback 进度回调
     * @param maxRetries 每个分片最大重试次数
     */
    Status uploadFileMultipart(const std::string& typeKey, 
                             const std::string& filePath,
                             const std::string& name,
                             std::function<void(double)> progressCallback = nullptr,
                             int maxRetries = 3);

    /**
     * 使用 STS 凭证进行 S3 原生上传 (需要 AWS SDK)
     * 注意：此方法仅在链接了 AWS SDK 的情况下可用，否则将抛出异常或返回失败。
     */
    Status uploadFileSTS(const UploadTicket& ticket, 
                       const std::string& filePath,
                       const std::string& endpoint = "localhost:9000");

private:
    std::unique_ptr<ClientImpl> impl_;
};

} // namespace simhub

#endif // SIMHUB_SDK_CLIENT_H
