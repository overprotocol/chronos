bazel build --remote_cache=http://192.168.2.200:9090 --stamp --config=minimal --config=windows_amd64 //cmd/beacon-chain //cmd/validator //cmd/prysmctl //tools/enr-calculator //tools/bootnode
if [ $? -ne 0 ]; then
    echo "Bazel build failed."
    exit 1
fi
zip -j dist/chronos_windows_minimal.zip bazel-bin/cmd/beacon-chain/beacon-chain_/beacon-chain.exe bazel-bin/cmd/validator/validator_/validator.exe bazel-bin/tools/enr-calculator/enr-calculator_/enr-calculator.exe bazel-bin/cmd/prysmctl/prysmctl_/prysmctl.exe bazel-bin/cmd/bootnode/bootnode_/bootnode.exe
rm -rf bazel-bin