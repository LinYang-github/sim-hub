cmake_minimum_required(VERSION 3.10)
project(SimHubSDK)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

# 依赖项查找
find_package(CURL REQUIRED)
option(USE_AWS_SDK "Build with AWS SDK support for STS uploads" OFF)

if(USE_AWS_SDK)
    find_package(AWSSDK REQUIRED COMPONENTS s3 core)
    add_definitions(-DUSE_AWS_SDK)
endif()

# 头文件目录
include_directories(include)
include_directories(${CURL_INCLUDE_DIRS})

# 源文件
set(SDK_SOURCES
    src/client.cpp
)

# 构建静态库
add_library(simhub_sdk STATIC ${SDK_SOURCES})

# 链接库
target_link_libraries(simhub_sdk PRIVATE ${CURL_LIBRARIES})

if(USE_AWS_SDK)
    target_link_libraries(simhub_sdk PRIVATE ${AWSSDK_LINK_LIBRARIES})
endif()

# 安装配置 (可选)
install(TARGETS simhub_sdk DESTINATION lib)
install(DIRECTORY include/simhub DESTINATION include)

# 示例程序 (可选开关)
# 示例程序 (可选开关)
option(BUILD_EXAMPLES "Build SDK examples" ON)
if(BUILD_EXAMPLES)
    add_executable(multipart_upload_example examples/multipart_upload.cpp)
    target_link_libraries(multipart_upload_example PRIVATE simhub_sdk ${CURL_LIBRARIES})
endif()

# --- 单元测试 / Unit Tests ---
option(BUILD_TESTS "Build SDK unit tests" ON)
if(BUILD_TESTS)
    include(FetchContent)
    FetchContent_Declare(
      googletest
      URL https://github.com/google/googletest/archive/refs/heads/main.zip
    )
    # For Windows: Prevent overriding the parent project's compiler/linker settings
    set(gtest_force_shared_crt ON CACHE BOOL "" FORCE)
    FetchContent_MakeAvailable(googletest)

    enable_testing()
    add_subdirectory(tests)
endif()
