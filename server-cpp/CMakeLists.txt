cmake_minimum_required(VERSION 4.0)
project(server_cpp)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
# disables certain symbols that were causing clangd to show errors in nvim
set(CMAKE_CXX_SCAN_FOR_MODULES OFF)
# export compile_commands.json that clangd can pickup to show definitions for api's
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

enable_testing()
add_subdirectory(src)
