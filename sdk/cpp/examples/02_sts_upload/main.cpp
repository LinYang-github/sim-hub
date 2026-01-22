#include <simhub/client.h>
#include <iostream>
#include <fstream>

#ifdef USE_AWS_SDK
#include <aws/core/Aws.h>
#endif

int main() {
#ifdef USE_AWS_SDK
    Aws::SDKOptions options;
    Aws::InitAPI(options);
    {
#endif
        simhub::Client client("http://localhost:30030");

        // 1. 请求 STS 凭证
        simhub::UploadTokenRequest req;
        req.resource_type = "scenario";
        req.filename = "sts_test.zip";
        req.mode = "sts";

        std::cout << "[STS] 正在请求凭证..." << std::endl;
        auto ticket = client.requestUploadToken(req);

        if (!ticket.has_credentials) {
            std::cerr << "[STS] 获取凭证失败" << std::endl;
            return 1;
        }

        // 2. 模拟本地文件
        std::string dummyFile = "sts_test.zip";
        std::ofstream out(dummyFile);
        out << "STS Upload Data via AWS SDK" << std::endl;
        out.close();

        // 3. 执行 STS 上传 (调用 AWS SDK)
        std::cout << "[STS] 正在通过 AWS SDK 上传至: " << ticket.bucket << "/" << ticket.object_key << std::endl;
        if (client.uploadFileSTS(ticket, dummyFile)) {
            std::cout << "[STS] 上传成功！" << std::endl;

            // 4. 确认流程
            simhub::ConfirmUploadRequest confirm;
            confirm.ticket_id = ticket.ticket_id;
            confirm.type_key = "scenario";
            confirm.name = "SDK_STS_Demo";
            confirm.owner_id = "sts_power_user";
            
            if (client.confirmUpload(confirm)) {
                std::cout << "[STS] 场景注册成功！" << std::endl;
            }
        } else {
            std::cerr << "[STS] 上传失败" << std::endl;
        }

#ifdef USE_AWS_SDK
    }
    Aws::ShutdownAPI(options);
#endif

    return 0;
}
