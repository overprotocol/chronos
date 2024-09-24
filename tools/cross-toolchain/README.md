# Multiarch Cross Compiling Toolchain

## Toolchain suite

This package declares a c++ toolchain suite with cross compilers for targeting five platforms:
* linux_amd64
* linux_arm64
* osx_amd64
* osx_arm64
* windows_amd64

This toolchain suite describes cross compile configuration with a Dockerfile with the appropriate host dependencies. These toolchains can be used locally (see [caveats](#caveats)), [Remote Build Execution (RBE)](https://docs.bazel.build/versions/master/remote-execution.html), and in a docker sandbox (like RBE, but local).

### Updating the toolchain suite

1) Build and push the worker docker image, if necessary.

```bash
docker buildx build --platform linux/amd64 -t overfoundation/bazel-cross:latest tools/cross-toolchain/.
docker push overfoundation/bazel-cross:latest
```

Note: You must configured your ghcr access credentials to push to OverFoundation Docker Hub. Contact SRE team for push access.

1) Note the docker image sha256 digest from the recently pushed image or use the latest one available.

2) Download and run [rbe_configs_gen](https://github.com/bazelbuild/bazel-toolchains#rbe_configs_gen---cli-tool-to-generate-configs) CLI tool.

```bash
# Run from the root of the Chronos repo.
rbe_configs_gen \
  --bazel_version=$(cat .bazelversion) \
  --target_os=linux \
  --exec_os=linux \
  --output_src_root=tools/cross-toolchain/configs \
  --generate_cpp_configs=true \
  --generate_java_configs=true \
  --cpp_env_json=tools/cross-toolchain/cpp_env_clang.json \
  --toolchain_container=overfoundation/bazel-cross@sha256:662e9fc2eee120b09fc67ad320a08cfab3609787827af005a0b9f49145cd1d6a # The sha256 digest from step 2.
```

4) Test the builds work locally for all supported platforms.

```bash
bazel build --config=release --config=linux_amd64 --config=llvm //cmd/beacon-chain //cmd/validator //cmd/client-stats //cmd/prysmctl
bazel build --config=release --config=linux_arm64_docker //cmd/beacon-chain //cmd/validator //cmd/client-stats //cmd/prysmctl
bazel build --config=release --config=osx_amd64_docker //cmd/beacon-chain //cmd/validator //cmd/client-stats //cmd/prysmctl
bazel build --config=release --config=osx_arm64_docker //cmd/beacon-chain //cmd/validator //cmd/client-stats //cmd/prysmctl 
bazel build --config=release --config=windows_amd64_docker //cmd/beacon-chain //cmd/validator //cmd/client-stats //cmd/prysmctl
```

5) Run gazelle.

```bash
bazel run //:gazelle
```

6) Add and commit the newly generated configs.

7) Done!

### Cross compile target support

| target           | linux_amd64                                                        | linux_arm64                                | osx_amd64                          | osx_arm64                          | windows_amd64                      |
| ---------------- | ------------------------------------------------------------------ | ------------------------------------------ | ---------------------------------- | ---------------------------------- | ---------------------------------- |
| `//beacon-chain` | :heavy_check_mark:  docker-sandbox and RBE, supported locally only | :heavy_check_mark:  docker-sandbox and RBE | :heavy_check_mark:  docker-sandbox | :heavy_check_mark:  docker-sandbox | :heavy_check_mark:  docker-sandbox |
| `//validator`    | :heavy_check_mark:  docker-sandbox and RBE                         | :heavy_check_mark: docker-sandbox and RBE  | :heavy_check_mark:  docker-sandbox | :heavy_check_mark:  docker-sandbox | :heavy_check_mark:                 |

The configurations above are enforced via pull request presubmit checks.

### Bazel config flag values

Use these values with `--config=<flag>`, multiple times if more than one value is defined in the table. Example: `bazel build //beacon-chain --config=windows_amd64_docker` to build windows binary in a docker sandbox.

| Config                        | linux_amd64                | linux_arm64                 | osx_amd64                 | osx_arm64                 | windows_amd64                 |
| ----------------------------- | -------------------------- | --------------------------- | ------------------------- | ------------------------- | ----------------------------- |
| Local run                     | `linux_amd64`              | `linux_arm64`               | `osx_amd64`               | `osx_arm64`               | `windows_amd64`               |
| Docker sandbox                | `linux_amd64_docker`       | `linux_arm64_docker`        | `osx_amd64_docker`        | `osx_arm64_docker`        | `windows_amd64_docker `       |
| RBE (See [Caveats](#caveats)) | `linux_amd64` and `remote` | `linux_arm64`  and `remote` | `osx_amd64`  and `remote` | `osx_arm64`  and `remote` | `windows_amd64`  and `remote` |

### Caveats

There are a few caveats to each of these strategies.

- Local runs require clang compiler and the appropriate cross compilers installed. These runs should only be considered for a power user or user with specific build requirements. See the Dockerfile setup scripts to understand what dependencies must be installed and where.
- Docker sandbox is *slow*. Like really slow! The purpose of the docker sandbox is to test RBE builds without deploying a full RBE system. Each build action is executed in its own container. Given the large number of small targets in this project, the overhead of creating docker containers makes this strategy the slowest of all, but requires zero additional setup.
- Remote Build Execution is by far the fastest, if you have a RBE backend available. This is another advanced use case which will require two config flags above as well as additional flags to specify the `--remote_executor`. Some of these flags are present in the project `.bazelrc` with example values, but commented out.