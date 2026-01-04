# TODO this script isn't working because it can't find boost. Could use clions cmake
# or do something else
cmake -B build \
    -DCMAKE_BUILD_TYPE=Debug \
    -DCMAKE_TOOLCHAIN_FILE=/home/rob/.vcpkg-clion/vcpkg/scripts/buildsystems/vcpkg.cmake &&
    cmake -B build -DCMAKE_BUILD_TYPE=Debug &&
    cmake --build cmake-build-debug/ --target AnAmazingBackend -j 30 &&
    cmake-build-debug/src/AnAmazingBackend
