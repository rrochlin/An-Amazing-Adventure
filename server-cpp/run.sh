#!/bin/bash
set -e

BUILD_DIR="build"

# Configure (only needs to re-run when CMakeLists.txt changes or new files are globbed)
cmake -B "$BUILD_DIR" \
    -DCMAKE_BUILD_TYPE=Debug \
    -DCMAKE_TOOLCHAIN_FILE=/home/rob/.vcpkg-clion/vcpkg/scripts/buildsystems/vcpkg.cmake

# Build
cmake --build "$BUILD_DIR" --target AnAmazingBackend -j 30

# Run
"./$BUILD_DIR/src/AnAmazingBackend"
