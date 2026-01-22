add_executable(demo_simple_upload main.cpp)
target_link_libraries(demo_simple_upload PRIVATE simhub_sdk)
target_include_directories(demo_simple_upload PRIVATE ../../include)
