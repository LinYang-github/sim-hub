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
    find_package(AWSSDK REQUIRED COMPONENTS s3 core)
endif()

# --- Library ---
# We provide a static library for those who want to link normally, 
# but the header is also usable as a Standalone Single-Header.
set(SDK_SOURCES
    src/simhub_impl.cpp
)

add_library(simhub_sdk STATIC ${SDK_SOURCES})
add_library(SimHub::SDK ALIAS simhub_sdk)

target_include_directories(simhub_sdk PUBLIC 
    $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
    $<INSTALL_INTERFACE:include>
)

target_link_libraries(simhub_sdk PRIVATE CURL::libcurl)

if(SIMHUB_ENABLE_AWS)
    target_compile_definitions(simhub_sdk PUBLIC USE_AWS_SDK)
    target_link_libraries(simhub_sdk PRIVATE ${AWSSDK_LINK_LIBRARIES})
endif()

# --- Examples ---
if(SIMHUB_BUILD_EXAMPLES)
    # 01: Quickstart
    add_executable(example_01_quickstart examples/01_quickstart/main.cpp)
    target_link_libraries(example_01_quickstart PRIVATE SimHub::SDK)

    # 02: Async Operations
    add_executable(example_02_async examples/02_async_ops/main.cpp)
    target_link_libraries(example_02_async PRIVATE SimHub::SDK)

    # 03: Bundle Download (requires C++17 filesystem)
    add_executable(example_03_bundle examples/03_bundle_download/main.cpp)
    target_link_libraries(example_03_bundle PRIVATE SimHub::SDK)

    # 04: Advanced Upload
    add_executable(example_04_upload examples/04_advanced_upload/main.cpp)
    target_link_libraries(example_04_upload PRIVATE SimHub::SDK)

    # 05: Multipart Concurrent Upload
    add_executable(example_05_multipart examples/05_multipart_upload/main.cpp)
    target_link_libraries(example_05_multipart PRIVATE SimHub::SDK)
endif()
# --- Tests ---
if(SIMHUB_BUILD_TESTS)
    enable_testing()
    find_package(GTest REQUIRED)
    include(GoogleTest)

    # Result/Types Test
    add_executable(test_simhub_types tests/test_result.cpp)
    target_link_libraries(test_simhub_types PRIVATE SimHub::SDK GTest::gtest_main)
    gtest_discover_tests(test_simhub_types)

    # Multipart Logic Test
    add_executable(test_simhub_multipart tests/test_multipart_logic.cpp)
    target_link_libraries(test_simhub_multipart PRIVATE SimHub::SDK GTest::gtest_main)
    gtest_discover_tests(test_simhub_multipart)
endif()

# --- Installation ---
include(GNUInstallDirs)
install(TARGETS simhub_sdk EXPORT SimHubSDKTargets DESTINATION ${CMAKE_INSTALL_LIBDIR})
install(DIRECTORY include/simhub DESTINATION ${CMAKE_INSTALL_INCLUDEDIR})
install(EXPORT SimHubSDKTargets FILE SimHubSDKTargets.cmake NAMESPACE SimHub:: DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/SimHubSDK)
