# Define the test executable
file(GLOB TEST_SOURCES "${CMAKE_CURRENT_SOURCE_DIR}/*.cpp")
add_executable(api_server_tests ${TEST_SOURCES})

# Link the library under test and GTest
target_link_libraries(api_server_tests PRIVATE
    api_server
    GTest::gtest
    GTest::gtest_main
)

# Register tests with CTest
gtest_discover_tests(api_server_tests)
