# OverProtocol Remote Build Execution Image with Cross Compile Support.
#
# See README.md for update instructions.

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

# install bazel
RUN npm install -g yarn @bazel/bazelisk

# install aws cli
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && unzip awscliv2.zip && ./aws/install

# install llvm/clang cross-build toolchains
ENV INSTALL_LLVM_VERSION=12.0.0
ADD install_clang_cross.sh /tmp/install_clang_cross.sh
RUN /tmp/install_clang_cross.sh

# install osxcross
ADD install_osxcross.sh /tmp/install_osxcross.sh
ADD common_osxcross.sh /tmp/common_osxcross.sh
RUN /tmp/install_osxcross.sh
ADD link_osxcross.sh /tmp/link_osxcross.sh
RUN /tmp/link_osxcross.sh