BASEDIR=$(pwd)
KAIROS_PATH=$BASEDIR/../../../kairos

bazel run --config=minimal //cmd/prysmctl:prysmctl testnet generate-genesis -- \
    -output-ssz=$BASEDIR/artifacts/genesis_minimal.ssz \
    -chain-config-file=$BASEDIR/artifacts/config_minimal.yml \
    -override-eth1data=true \
    -geth-genesis-json-in=$KAIROS_PATH/testnet/under/artifacts/genesis.json \
    -deposit-json-file=$BASEDIR/artifacts/deposits/deposit_data_under.json \
    -fork=phase0 \
    -num-validators=0