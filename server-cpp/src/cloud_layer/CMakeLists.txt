add_library(cloud_layer STATIC src/main.cpp)

target_include_directories(cloud_layer PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include/cloud_layer>
)