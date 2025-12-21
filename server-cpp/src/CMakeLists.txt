find_package(Boost REQUIRED COMPONENTS uuid beast date_time)
find_package(nlohmann_json CONFIG REQUIRED)
find_package(AWSSDK CONFIG COMPONENTS dynamodb s3 REQUIRED)
#find_package(google_cloud_cpp_{this will be unique per package} CONFIG REQUIRED)


add_subdirectory(game_engine)
add_subdirectory(api_server)
add_subdirectory(cloud_layer)

add_executable(AnAmazingBackend main.cpp)

target_link_libraries(AnAmazingBackend PRIVATE nlohmann_json::nlohmann_json)

target_link_libraries(AnAmazingBackend PRIVATE
game_engine
api_server
cloud_layer
)





