cmake_minimum_required(VERSION 3.10)
project(SimHubSTSExample)

set(CMAKE_CXX_STANDARD 17)

# Find AWS SDK
find_package(AWSSDK REQUIRED COMPONENTS s3 core)

# Find Curl
find_package(CURL REQUIRED)

include_directories(${CURL_INCLUDE_DIRS})
include_directories(.) 

add_executable(sts_example main.cpp)

target_link_libraries(sts_example 
    ${AWSSDK_LINK_LIBRARIES}
    ${CURL_LIBRARIES}
)
