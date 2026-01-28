cmake_minimum_required(VERSION 3.14)
project(SimHubSDK VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

# --- Options ---
option(SIMHUB_ENABLE_AWS "Enable AWS SDK support for STS direct uploads" OFF)
option(SIMHUB_BUILD_EXAMPLES "Build example applications" ON)
option(SIMHUB_BUILD_TESTS "Build unit tests" OFF)

# --- Dependencies ---
find_package(CURL REQUIRED)

if(SIMHUB_ENABLE_AWS)
    # Hint: You might need to set -DCMAKE_PREFIX_PATH=/path/to/aws/sdk
    find_package(AWSSDK REQUIRED COMPONENTS s3 core)
endif()

# --- Library ---
set(SDK_SOURCES
    src/client.cpp
)

set(SDK_HEADERS
    include/simhub/simhub.hpp
)

add_library(simhub_sdk STATIC ${SDK_SOURCES})
add_library(SimHub::SDK ALIAS simhub_sdk)

# Include Directories
target_include_directories(simhub_sdk PUBLIC 
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:include>
)

# Compile Definitions & Links
target_link_libraries(simhub_sdk PRIVATE CURL::libcurl)

if(SIMHUB_ENABLE_AWS)
    target_compile_definitions(simhub_sdk PUBLIC USE_AWS_SDK)
    target_link_libraries(simhub_sdk PRIVATE ${AWSSDK_LINK_LIBRARIES})
    message(STATUS "SimHub SDK: AWS Support Enabled")
else()
    message(STATUS "SimHub SDK: AWS Support Disabled (HTTP Only)")
endif()

# --- Installation ---
include(GNUInstallDirs)
install(TARGETS simhub_sdk
    EXPORT SimHubSDKTargets
    LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
    ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR}
)

install(DIRECTORY include/simhub DESTINATION ${CMAKE_INSTALL_INCLUDEDIR})

# Export Config
install(EXPORT SimHubSDKTargets
    FILE SimHubSDKTargets.cmake
    NAMESPACE SimHub::
    DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/SimHubSDK
)

# --- Examples ---
if(SIMHUB_BUILD_EXAMPLES)
    # Simple multipart example
    if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/examples/multipart_upload.cpp")
        add_executable(multipart_example examples/multipart_upload.cpp)
        target_link_libraries(multipart_example PRIVATE SimHub::SDK)
    endif()
    
    # Single header demo (if we want to include it here, or let it be separate)
    if(EXISTS "${CMAKE_CURRENT_SOURCE_DIR}/examples/single_header_demo/main.cpp")
         add_executable(single_header_demo examples/single_header_demo/main.cpp)
         target_link_libraries(single_header_demo PRIVATE SimHub::SDK)
    endif()
endif()
