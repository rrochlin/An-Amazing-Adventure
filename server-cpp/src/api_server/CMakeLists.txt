add_library(api_server STATIC src/server.cpp src/listener.cpp src/session.cpp src/server_utils.cpp)

target_include_directories(api_server PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include/api_server>
)

target_link_libraries(api_server
    PUBLIC Boost::beast
    PRIVATE Boost::asio
)
