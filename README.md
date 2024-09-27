# Chronos: An Over Protocol Consensus Implementation Written in Go

Official Go implementation of the [Over protocol](https://over.network/) consensus layer. Chronos is a fork of [Prysm](https://github.com/prysmaticlabs/prysm), a [Golang](https://golang.org/) implementation of the [Ethereum Consensus](https://ethereum.org/en/developers/docs/consensus-mechanisms/#proof-of-stake) specification.

## Getting Started

A detailed set of installation and usage instructions, as well as breakdowns of each individual component, refer to [Prysm's documentation portal](https://docs.prylabs.network). If you still have questions, feel free to stop by our [Discord](https://discord.com/invite/overprotocol).

### Building the Source

Chronos can be installed using [Bazel](https://bazel.build/). You can install Bazel using Bazelisk, a handy tool for launching Bazel. Please refer to [Bazelisk official repository](https://github.com/bazelbuild/bazelisk?tab=readme-ov-file#installation) for installation instructions.

Once the dependencies, including Bazel, are installed, run the following command:

```shell
bazel build //cmd/beacon-chain:beacon-chain //cmd/validator:validator
```

Bazel will automatically generate symlinks at `bazel-bin/`. You can find the beacon-chain binary at `bazel-bin/cmd/beacon-chain/beacon-chain_/beacon-chain`, and the `validator` binary likewise.

### Operating Validators

To operate validators with Chronos, follow the steps outlined in [our official documentation](https://docs.over.network/operators/operate-validators).

## Contributing

We welcome contributions from the community. Please refer to our [contributing guidelines](CONTRIBUTING.md) to get started.

## License

[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html)
