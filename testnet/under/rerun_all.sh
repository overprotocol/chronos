BASEDIR=$(pwd)
KAIROS_PATH=$BASEDIR/../../../kairos

# Get the OS name
os_name=$(uname)

minimal=""
minimal_suffix=""

tcpport_base=13000

if [ "$1" = "minimal" ]; then
    echo "minimal build"
    minimal="--config=minimal "
    minimal_suffix="_minimal"
fi

# Kill chronos nodes if exists.
for i in $(seq 0 1); do
    # change these to the unique parts of your command
    unique_part="p2p-tcp-port=$(($tcpport_base + i))"

    pids=$(ps aux | grep "${unique_part}" | grep -v grep | awk '{print $2}')

    if [ -z "$pids" ]
    then
        echo "No processes found with command parts $unique_part"
    else
        echo "Killing Chronos processes with PIDs: $pids"
        for pid in $pids
        do
            kill -9 $pid
        done
    fi
done

# Kill validator clients if exists.
validator_part="wallet-dir=$BASEDIR/validator-"

pids=$(ps aux | grep "${validator_part}" | grep -v grep | awk '{print $2}')

if [ -z "$pids" ]
then
    echo "No validator client processes found"
else
    echo "Killing Validator processes with PIDs: $pids"
    for pid in $pids
    do
        kill -9 $pid
    done
fi

if [ "$1" = "stop" ]; then
    exit 0
fi

# Replace Genesis timestamp for new beacon chain
current_date=$(date +%s)
target_date=$((current_date + 60))

if [ "$os_name" = "Linux" ]; then
    echo "The running machine is Linux."
    echo "Target genesis time updated to : $(date -d @$target_date)"
elif [ "$os_name" = "Darwin" ]; then
    echo "The running machine is macOS."
    echo "Target genesis time updated to : $(date -r $target_date)"
else
    echo "The running machine is neither Linux nor macOS. So there can be some problems."
fi

bazel run $minimal//tools/change-genesis-timestamp -- \
    -genesis-state=$BASEDIR/artifacts/genesis$minimal_suffix.ssz \
    -timestamp=$target_date

# Reset and run chronos nodes and validator clients.
rm -rf $BASEDIR/logs/chronos-*
rm -rf $BASEDIR/logs/vali-*

for i in $(seq 0 1); do
    rm -rf $BASEDIR/node-$i/beaconchaindata
    rm -rf $BASEDIR/node-$i/metaData

    nohup $BASEDIR/node-$i/run_node.sh > logs/chronos-$i.out &
    echo "Chronos-$i node started"
    nohup $BASEDIR/validator-$i/run_validator.sh > logs/vali-$i.out &
    echo "Validator-$i client started"
done