bazel build --remote_cache=http://192.168.2.200:9090 --stamp --config=minimal --config=linux_arm64 //cmd/beacon-chain //cmd/validator //cmd/prysmctl //tools/enr-calculator //tools/bootnode
if [ $? -ne 0 ]; then
    echo "Bazel build failed."
    exit 1
fi
zip -j dist/chronos_linux_arm64_minimal.zip bazel-bin/cmd/beacon-chain/beacon-chain_/beacon-chain bazel-bin/cmd/validator/validator_/validator bazel-bin/tools/enr-calculator/enr-calculator_/enr-calculator bazel-bin/cmd/prysmctl/prysmctl_/prysmctl bazel-bin/tools/bootnode/bootnode_/bootnode
rm -rf bazel-bin