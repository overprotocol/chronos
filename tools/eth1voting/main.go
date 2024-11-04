package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"time"

	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	v1alpha1 "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var (
	beacon  = flag.String("beacon", "127.0.0.1:4000", "gRPC address of the Prysm beacon node")
	genesis = flag.Uint64("genesis", 1606824023, "Genesis time. mainnet=1606824023, holesky=1695902400")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	cc, err := grpc.DialContext(ctx, *beacon, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt64)))
	if err != nil {
		panic(err)
	}
	c := v1alpha1.NewBeaconChainClient(cc)
	g, ctx := errgroup.WithContext(ctx)
	v := newVotes()

	current := slots.ToEpoch(slots.CurrentSlot(*genesis))
	start := current.Div(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod)).Mul(uint64(params.BeaconConfig().EpochsPerEth1VotingPeriod))
	nextStart := start.AddEpoch(params.BeaconConfig().EpochsPerEth1VotingPeriod)

	fmt.Printf("Looking back from current epoch %d back to %d\n", current, start)
	nextStartSlot, err := slots.EpochStart(nextStart)
	if err != nil {
		panic(err)
	}
	nextStartTime, err := slots.ToTime(*genesis, nextStartSlot)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Next period starts at epoch %d (%s)\n", nextStart, time.Until(nextStartTime))

	for i := primitives.Epoch(0); i < current.Sub(uint64(start)); i++ {
		j := i
		g.Go(func() error {
			_, err := c.ListBeaconBlocks(ctx, &v1alpha1.ListBlocksRequest{
				QueryFilter: &v1alpha1.ListBlocksRequest_Epoch{Epoch: current.Sub(uint64(j))},
			})
			if err != nil {
				return err
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		panic(err)
	}

	fmt.Println(v.Report())
}
