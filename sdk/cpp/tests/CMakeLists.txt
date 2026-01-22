add_executable(sdk_tests
    test_result.cpp
)

add_executable(sdk_stress_test
    stress_test.cpp
)

target_link_libraries(sdk_tests
    GTest::gtest_main
    simhub_sdk
)

target_link_libraries(sdk_stress_test
    simhub_sdk
    pthread
)

include(GoogleTest)
gtest_discover_tests(sdk_tests)
