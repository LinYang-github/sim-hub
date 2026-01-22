add_executable(demo_sts_upload main.cpp)
target_link_libraries(demo_sts_upload PRIVATE simhub_sdk)
target_include_directories(demo_sts_upload PRIVATE ../../include)

if(USE_AWS_SDK)
    target_compile_definitions(demo_sts_upload PRIVATE USE_AWS_SDK)
endif()
