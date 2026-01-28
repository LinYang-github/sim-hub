cmake_minimum_required(VERSION 3.10)
project(simhub_demo)

set(CMAKE_CXX_STANDARD 17)

# --- Dependencies ---
find_package(CURL REQUIRED)

# --- Option to enable AWS for this demo ---
option(USE_AWS_SDK "Enable AWS SDK support" OFF)

if(USE_AWS_SDK)
    find_package(AWSSDK REQUIRED COMPONENTS s3 core)
endif()

# --- Include Directories ---
include_directories(../../include)

# --- Application ---
add_executable(demo main.cpp ../../src/client.cpp)
target_link_libraries(demo CURL::libcurl)

if(USE_AWS_SDK)
    target_link_libraries(demo ${AWSSDK_LINK_LIBRARIES})
    target_compile_definitions(demo PRIVATE USE_AWS_SDK)
    message(STATUS "Demo building WITH AWS SDK support")
else()
    message(STATUS "Demo building WITHOUT AWS SDK support")
endif()
