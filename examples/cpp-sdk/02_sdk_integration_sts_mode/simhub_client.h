#ifndef SIMHUB_CLIENT_H
#define SIMHUB_CLIENT_H

#include <string>
#include <vector>
#include <iostream>
#include <fstream>
#include <sys/stat.h>
#include <curl/curl.h>
#include "json.hpp"

using json = nlohmann::json;

class SimHubClient {
private:
    std::string baseUrl;
    
    static size_t WriteCallback(void* contents, size_t size, size_t nmemb, void* userp) {
        ((std::string*)userp)->append((char*)contents, size * nmemb);
        return size * nmemb;
    }

public:
    SimHubClient(std::string url) : baseUrl(url) {}

    // POST Request
    json post(std::string endpoint, json body) {
        CURL* curl;
        CURLcode res;
        std::string readBuffer;

        curl = curl_easy_init();
        if(curl) {
            std::string url = baseUrl + endpoint;
            std::cout << "DEBUG: URL = [" << url << "]" << std::endl;
            curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
            
            std::string jsonStr = body.dump();
            curl_easy_setopt(curl, CURLOPT_POSTFIELDS, jsonStr.c_str());

            struct curl_slist* headers = NULL;
            headers = curl_slist_append(headers, "Content-Type: application/json");
            curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
            
            curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
            curl_easy_setopt(curl, CURLOPT_WRITEDATA, &readBuffer);

            long http_code = 0;
            res = curl_easy_perform(curl);
            curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &http_code);
            
            if(res != CURLE_OK) {
                // Use cout instead of stderr to be visible in artifacts
                std::cout << "curl_easy_perform() failed: " << curl_easy_strerror(res) << ", Code: " << res << std::endl;
                return json{{"error", "Network error"}};
            }

            curl_easy_cleanup(curl);
            curl_slist_free_all(headers);

            try {
                return json::parse(readBuffer);
            } catch (const std::exception& e) {
                std::cerr << "JSON Parse Error on: " << readBuffer << std::endl;
                return json{{"error", "Failed to parse response: " + readBuffer}};
            }
        }
        return json{{"error", "Failed to init curl"}};
    }

    // PUT File (Upload)
    bool uploadFile(std::string url, std::string filePath) {
        CURL* curl;
        CURLcode res;
        FILE *fd;
        struct stat file_info;

        fd = fopen(filePath.c_str(), "rb");
        if(!fd) return false;

        fstat(fileno(fd), &file_info);
        curl_off_t fileSize = file_info.st_size;

        curl = curl_easy_init();
        if(curl) {
            curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
            curl_easy_setopt(curl, CURLOPT_UPLOAD, 1L);
            curl_easy_setopt(curl, CURLOPT_READDATA, fd);
            curl_easy_setopt(curl, CURLOPT_INFILESIZE_LARGE, fileSize);
            
            res = curl_easy_perform(curl);
            
            curl_easy_cleanup(curl);
        }
        fclose(fd);
        return res == CURLE_OK;
    }
    
    // Helper: Upload Scenario
    void uploadScenario(std::string name, std::string zipPath) {
        std::cout << "[Step 1] Requesting Upload Token for: " << name << std::endl;
        
        // 1. Get Token
        json reqToken = {
            {"resource_type", "scenario"},
            {"filename", name + ".zip"},
            {"size", 0}, // Optional in MVP
            {"checksum", ""}
        };
        
        json resToken = post("/api/v1/integration/upload/token", reqToken);
        if (resToken.contains("error")) {
            std::cerr << "Error getting token: " << resToken["error"] << std::endl;
            return;
        }

        std::string ticketId = resToken["ticket_id"];
        std::string presignedUrl = resToken["presigned_url"];
        
        std::cout << "[Step 2] Uploading file to storage..." << std::endl;
        
        // 2. Upload to MinIO
        if (!uploadFile(presignedUrl, zipPath)) {
            std::cerr << "Failed to upload file." << std::endl;
            return;
        }
        std::cout << "Upload completed." << std::endl;

        // 3. Confirm
        std::cout << "[Step 3] Confirming upload..." << std::endl;
        json reqConfirm = {
            {"ticket_id", ticketId},
            {"type_key", "scenario"},
            {"name", name},
            {"owner_id", "cpp-client"},
            {"size", 1024}, // Mock size
            {"extra_meta", {{"source", "cpp-sdk"}}}
        };
        
        json resConfirm = post("/api/v1/integration/upload/confirm", reqConfirm);
        if (resConfirm.contains("code") && resConfirm["code"] == 200) {
            std::cout << "Success! Scenario registered." << std::endl;
        } else {
            std::cerr << "Confirmation failed." << std::endl;
        }
    }
};

#endif
