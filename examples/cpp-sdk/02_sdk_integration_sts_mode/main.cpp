#include <iostream>
#include <fstream>
#include <vector>
#include <sys/stat.h>
#include "simhub_client.h"

// AWS SDK 头文件
#include <aws/core/Aws.h>
#include <aws/core/auth/AWSCredentials.h>
#include <aws/s3/S3Client.h>
#include <aws/s3/model/PutObjectRequest.h>
#include <aws/core/utils/memory/stl/AWSStringStream.h>

int main(int argc, char* argv[]) {
    // 1. 准备测试文件
    std::string zipPath = "test.zip";
    if (argc >= 2) zipPath = argv[1];
    
    // 确保文件存在
    struct stat buffer;
    if (stat(zipPath.c_str(), &buffer) != 0) {
        std::ofstream outfile(zipPath);
        outfile << "Real STS Upload Content via AWS SDK" << std::endl;
        outfile.close();
        std::cout << "已创建虚拟文件 " << zipPath << std::endl;
    }

    SimHubClient client("http://localhost:30030");
    std::string name = "Real_AWS_SDK_Test";
    
    // 2. 初始化 AWS SDK 全局环境
    Aws::SDKOptions options;
    Aws::InitAPI(options);
    { // AWS 对象的作用域
        // 3. 从 SimHub 请求 STS 凭证
        std::cout << "[步骤 1] 正在请求 STS 凭证..." << std::endl;
        json reqToken = {
            {"resource_type", "scenario"},
            {"filename", name + ".zip"},
            {"size", 1024},
            {"checksum", "none"},
            {"mode", "sts"} 
        };

        json res = client.post("/api/v1/integration/upload/token", reqToken);

        if (res.contains("error")) {
            std::cerr << "错误: " << res["error"] << std::endl;
            return 1;
        }

        json creds = res["credentials"];
        std::string ak = creds["access_key"];
        std::string sk = creds["secret_key"];
        std::string token = creds["session_token"];
        std::string bucket = res["bucket"];
        std::string objectKey = res["object_key"];
        std::string ticketId = res["ticket_id"];

        std::cout << "[步骤 2] 已获取凭证，目标: " << bucket << "/" << objectKey << std::endl;

        // 4. 使用临时凭证配置 S3 客户端
        Aws::Auth::AWSCredentials awsCreds(ak.c_str(), sk.c_str(), token.c_str());
        Aws::Client::ClientConfiguration clientConfig;
        clientConfig.endpointOverride = "localhost:9000"; // 指向本地 MinIO
        clientConfig.scheme = Aws::Http::Scheme::HTTP;
        clientConfig.verifySSL = false; // 用于本地开发环境
        
        // 针对 MinIO 的通常要求，默认使用路径样式访问
        // AWS SDK 1.9+ 默认使用虚拟主机样式
        // 这里使用标准的 S3Client 配置
        Aws::S3::S3Client s3_client(awsCreds, clientConfig, 
            Aws::Client::AWSAuthV4Signer::PayloadSigningPolicy::Never, // 在 HTTP 环境下，MinIO 有时不需要负载签名
            false); // useVirtualAddressing = false (路径样式)

        // 5. 上传文件
        std::cout << "[步骤 3] 正在通过 AWS SDK 上传文件..." << std::endl;
        Aws::S3::Model::PutObjectRequest request;
        request.SetBucket(bucket.c_str());
        request.SetKey(objectKey.c_str());

        std::shared_ptr<Aws::IOStream> inputData = Aws::MakeShared<Aws::FStream>("SampleAllocationTag",
            zipPath.c_str(), 
            std::ios_base::in | std::ios_base::binary);

        request.SetBody(inputData);

        auto outcome = s3_client.PutObject(request);

        if (!outcome.IsSuccess()) {
             std::cerr << "S3 上传错误: " << outcome.GetError().GetMessage() << std::endl;
             // 此处不立即返回 1，继续尝试确认流程
        } else {
             std::cout << "S3 上传成功！" << std::endl;
        }

        // 6. 向 SimHub 确认上传
        if (outcome.IsSuccess()) {
            std::cout << "\n[步骤 4] 正在确认上传..." << std::endl;
            json reqConfirm = {
                {"ticket_id", ticketId},
                {"type_key", "scenario"},
                {"name", name},
                {"owner_id", "aws-sdk-cpp-client"},
                {"size", 1024}, 
                {"extra_meta", {{"method", "aws_sdk_cpp"}}}
            };
            
            json resConfirm = client.post("/api/v1/integration/upload/confirm", reqConfirm);
            if (resConfirm.contains("error")) {
                std::cerr << "确认失败: " << resConfirm["error"] << std::endl;
            } else {
                std::cout << "成功！场景已通过真实 AWS SDK 注册。" << std::endl;
            }
        }

    } // AWS 作用域结束
    Aws::ShutdownAPI(options);
    
    return 0;
}
