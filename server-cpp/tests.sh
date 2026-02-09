#!/bin/bash
set -e

BUILD_DIR="build"

cmake -B "$BUILD_DIR" \
    -DCMAKE_BUILD_TYPE=Debug \
    -DCMAKE_TOOLCHAIN_FILE=/home/rob/.vcpkg-clion/vcpkg/scripts/buildsystems/vcpkg.cmake >/dev/null

cmake --build "$BUILD_DIR" -j 30 >build.log

cd "$BUILD_DIR"
ctest --output-on-failure
