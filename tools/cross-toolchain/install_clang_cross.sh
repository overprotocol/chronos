#!/bin/sh

set -eu

# Download and install LLVM for x86_64
curl -L https://github.com/llvm/llvm-project/releases/download/llvmorg-"${INSTALL_LLVM_VERSION}"/clang+llvm-"${INSTALL_LLVM_VERSION}"-x86_64-linux-gnu-ubuntu-20.04.tar.xz \
    -o clang+llvm-"${INSTALL_LLVM_VERSION}"-x86_64-linux-gnu-ubuntu-20.04.tar.xz \
    && tar xf clang+llvm-"${INSTALL_LLVM_VERSION}"-x86_64-linux-gnu-ubuntu-20.04.tar.xz --strip-components=1 -C /usr \
    && rm -f clang+llvm-"${INSTALL_LLVM_VERSION}"-x86_64-linux-gnu-ubuntu-20.04.tar.xz

# Download and install LLVM for aarch64
curl -L https://github.com/llvm/llvm-project/releases/download/llvmorg-"${INSTALL_LLVM_VERSION}"/clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu.tar.xz \
    -o clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu.tar.xz \
    && tar xf clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu.tar.xz \
    && rm -f clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu.tar.xz

# Move necessary files for aarch64 to the correct locations
mkdir -p /usr/aarch64-linux-gnu/lib/clang/"${INSTALL_LLVM_VERSION}"
mkdir -p /usr/aarch64-linux-gnu/include/c++

mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/include/c++/v1 /usr/aarch64-linux-gnu/include/c++/
mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/clang/"${INSTALL_LLVM_VERSION}"/include /usr/aarch64-linux-gnu/lib/clang/"${INSTALL_LLVM_VERSION}"

mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/libc++.a /usr/aarch64-linux-gnu/lib/
mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/libc++abi.a /usr/aarch64-linux-gnu/lib/
mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/libunwind.a /usr/aarch64-linux-gnu/lib/

mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/clang/"${INSTALL_LLVM_VERSION}"/lib/linux/libclang_rt.builtins-aarch64.a /usr/lib/clang/"${INSTALL_LLVM_VERSION}"/lib/linux/
mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/clang/"${INSTALL_LLVM_VERSION}"/lib/linux/clang_rt.crtbegin-aarch64.o /usr/lib/clang/"${INSTALL_LLVM_VERSION}"/lib/linux/
mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/clang/"${INSTALL_LLVM_VERSION}"/lib/linux/clang_rt.crtend-aarch64.o /usr/lib/clang/"${INSTALL_LLVM_VERSION}"/lib/linux/

# Clean up
rm -rf /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu