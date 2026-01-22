#include <simhub/client.h>
#include <iostream>
#include <fstream>

int main() {
    // 全局初始化 (必须只需调用一次)
    simhub::Client::GlobalInit();

    simhub::Client client("http://localhost:30030");

    // 1. 准备上传请求 (默认模式为 presigned)
    simhub::UploadTokenRequest req;
    req.resource_type = "scenario";
    req.filename = "simple_test.zip";
    req.mode = "presigned";

    std::cout << "正在请求上传令牌..." << std::endl;
    auto ticket = client.requestUploadToken(req);

    if (ticket.ticket_id.empty()) {
        std::cerr << "申请令牌失败" << std::endl;
        return 1;
    }

    // 2. 模拟本地压缩包
    std::string dummyFile = "simple_test.zip";
    std::ofstream out(dummyFile);
    out << "Simple Upload Data" << std::endl;
    out.close();

    // 3. 执行上传
    std::cout << "正在上传文件: " << ticket.presigned_url << std::endl;
    bool success = client.uploadFileSimple(ticket.presigned_url, dummyFile, [](double p){
        std::cout << "\r上传进度: " << (int)(p * 100) << "%" << std::flush;
    });
    std::cout << std::endl;

    if (!success) {
        std::cerr << "上传失败" << std::endl;
        return 1;
    }

    // 4. 确认上传
    simhub::ConfirmUploadRequest confirm;
    confirm.ticket_id = ticket.ticket_id;
    confirm.type_key = "scenario";
    confirm.name = "SDK_Simple_Demo";
    confirm.owner_id = "cpp_sdk_user";
    
    if (client.confirmUpload(confirm)) {
        std::cout << "上传并确认成功！" << std::endl;
    } else {
        std::cerr << "确认失败" << std::endl;
    }

    simhub::Client::GlobalCleanup();
    return 0;
}
