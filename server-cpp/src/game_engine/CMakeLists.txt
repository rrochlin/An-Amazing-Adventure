add_library(game_engine STATIC src/main.cpp)

target_include_directories(game_engine PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include/game_engine>
)

