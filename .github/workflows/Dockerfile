FROM mcr.microsoft.com/dotnet/runtime-deps:6.0-jammy

# install gnu/gcc cross-build toolchain (gcc 8.3)
RUN apt-get update -y \
    && apt-get install -y \
    sudo lsb-release nodejs npm libtool libyaml-dev unzip curl git gcc-multilib xz-utils gcc g++ mingw-w64 \
    cmake libssl-dev libxml2-dev vim apt-transport-https \
    zip libtinfo5 patch zlib1g-dev autoconf pkg-config make docker.io gnupg2 libgmp-dev python3 \
    openjdk-17-jdk-headless tini \
    && rm -rf /var/lib/apt/lists/*
RUN apt-get update -y \
    && apt-get install -y \
    gcc-aarch64-linux-gnu g++-aarch64-linux-gnu \
    && rm -rf /var/lib/apt/lists/*
RUN npm install -g yarn @bazel/bazelisk
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && unzip awscliv2.zip && ./aws/install

# install llvm/clang cross-build toolchains
ENV INSTALL_LLVM_VERSION=12.0.0
RUN curl -L https://github.com/llvm/llvm-project/releases/download/llvmorg-"${INSTALL_LLVM_VERSION}"/clang+llvm-"${INSTALL_LLVM_VERSION}"-x86_64-linux-gnu-ubuntu-20.04.tar.xz \
    -o clang+llvm-"${INSTALL_LLVM_VERSION}"-x86_64-linux-gnu-ubuntu-20.04.tar.xz \
    && tar xf clang+llvm-"${INSTALL_LLVM_VERSION}"-x86_64-linux-gnu-ubuntu-20.04.tar.xz --strip-components=1 -C /usr \
    && rm -f clang+llvm-"${INSTALL_LLVM_VERSION}"-x86_64-linux-gnu-ubuntu-20.04.tar.xz
RUN curl -L https://github.com/llvm/llvm-project/releases/download/llvmorg-"${INSTALL_LLVM_VERSION}"/clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu.tar.xz \
    -o clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu.tar.xz \
    && tar xf clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu.tar.xz \
    && rm -f clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu.tar.xz \
    && mkdir -p /usr/aarch64-linux-gnu/lib/clang/"${INSTALL_LLVM_VERSION}" \
    && mkdir -p /usr/aarch64-linux-gnu/include/c++ \
    && mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/include/c++/v1 /usr/aarch64-linux-gnu/include/c++/ \
    && mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/clang/"${INSTALL_LLVM_VERSION}"/include /usr/aarch64-linux-gnu/lib/clang/"${INSTALL_LLVM_VERSION}" \
    && mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/libc++.a /usr/aarch64-linux-gnu/lib/ \
    && mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/libc++abi.a /usr/aarch64-linux-gnu/lib/ \
    && mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/libunwind.a /usr/aarch64-linux-gnu/lib/ \
    && mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/clang/"${INSTALL_LLVM_VERSION}"/lib/linux/libclang_rt.builtins-aarch64.a /usr/lib/clang/"${INSTALL_LLVM_VERSION}"/lib/linux/ \
    && mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/clang/"${INSTALL_LLVM_VERSION}"/lib/linux/clang_rt.crtbegin-aarch64.o /usr/lib/clang/"${INSTALL_LLVM_VERSION}"/lib/linux/ \
    && mv /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu/lib/clang/"${INSTALL_LLVM_VERSION}"/lib/linux/clang_rt.crtend-aarch64.o /usr/lib/clang/"${INSTALL_LLVM_VERSION}"/lib/linux/ \
    && rm -rf /clang+llvm-"${INSTALL_LLVM_VERSION}"-aarch64-linux-gnu

# install osxcross
ARG OSXCROSS_REPO=tpoechtrager/osxcross
ARG OSXCROSS_SHA1=17bb5e2d0a46533c1dd525cf4e9a80d88bd9f00e
ARG OSX_SDK=MacOSX12.3.sdk
ARG OSX_SDK_SUM=ae92043c991873bbecd716f8a3ea1d1623698dc556843d590d7ca9be76f28414
RUN mkdir -p /usr/osxcross \
    && mkdir -p /tmp/osxcross
WORKDIR /tmp/osxcross
RUN curl -sLo osxcross.tar.gz https://codeload.github.com/${OSXCROSS_REPO}/tar.gz/${OSXCROSS_SHA1} \
    && tar --strip=1 -xzf osxcross.tar.gz \
    && rm -f osxcross.tar.gz
RUN curl -sLo tarballs/${OSX_SDK}.tar.xz https://prysmaticlabs.com/uploads/${OSX_SDK}.tar.xz \
    && echo "${OSX_SDK_SUM}"  tarballs/${OSX_SDK}.tar.xz | sha256sum -c - \
    && yes "" | SDK_VERSION=12.3 OSX_VERSION_MIN=11.0 OCDEBUG=1 ./build.sh
ADD common_osxcross.sh /tmp/common_osxcross.sh
ADD link_osxcross.sh /tmp/link_osxcross.sh
RUN /tmp/link_osxcross.sh
