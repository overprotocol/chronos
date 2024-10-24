package validator

import (
	"context"
	"testing"

	chainMock "github.com/prysmaticlabs/prysm/v5/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/block"
	statefeed "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/mock"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"go.uber.org/mock/gomock"
)

func TestServer_StreamAltairBlocksVerified_ContextCanceled(t *testing.T) {
	ctx := context.Background()

	chainService := &chainMock.ChainService{}
	ctx, cancel := context.WithCancel(ctx)
	server := &Server{
		Ctx:           ctx,
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidatorAltair_StreamBlocksServer(ctrl)
	mockStream.EXPECT().Context().Return(ctx)
	go func(tt *testing.T) {
		assert.ErrorContains(tt, "Context canceled", server.StreamBlocksAltair(&ethpb.StreamBlocksRequest{
			VerifiedOnly: true,
		}, mockStream))
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestServer_StreamAltairBlocks_ContextCanceled(t *testing.T) {
	ctx := context.Background()

	chainService := &chainMock.ChainService{}
	ctx, cancel := context.WithCancel(ctx)
	server := &Server{
		Ctx:           ctx,
		BlockNotifier: chainService.BlockNotifier(),
		HeadFetcher:   chainService,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidatorAltair_StreamBlocksServer(ctrl)
	mockStream.EXPECT().Context().Return(ctx)
	go func(tt *testing.T) {
		assert.ErrorContains(tt, "Context canceled", server.StreamBlocksAltair(&ethpb.StreamBlocksRequest{}, mockStream))
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestServer_StreamSlotsVerified_ContextCanceled(t *testing.T) {
	ctx := context.Background()

	chainService := &chainMock.ChainService{}
	ctx, cancel := context.WithCancel(ctx)
	server := &Server{
		Ctx:           ctx,
		StateNotifier: chainService.StateNotifier(),
		HeadFetcher:   chainService,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_StreamSlotsServer(ctrl)
	mockStream.EXPECT().Context().Return(ctx)
	go func(tt *testing.T) {
		assert.ErrorContains(tt, "Context canceled", server.StreamSlots(&ethpb.StreamSlotsRequest{
			VerifiedOnly: true,
		}, mockStream))
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestServer_StreamSlots_ContextCanceled(t *testing.T) {
	ctx := context.Background()

	chainService := &chainMock.ChainService{}
	ctx, cancel := context.WithCancel(ctx)
	server := &Server{
		Ctx:           ctx,
		BlockNotifier: chainService.BlockNotifier(),
		HeadFetcher:   chainService,
	}

	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_StreamSlotsServer(ctrl)
	mockStream.EXPECT().Context().Return(ctx)
	go func(tt *testing.T) {
		assert.ErrorContains(tt, "Context canceled", server.StreamSlots(&ethpb.StreamSlotsRequest{}, mockStream))
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestServer_StreamSlots_OnHeadUpdated(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.BeaconConfig())
	ctx := context.Background()

	chainService := &chainMock.ChainService{}
	server := &Server{
		Ctx:           ctx,
		BlockNotifier: chainService.BlockNotifier(),
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_StreamSlotsServer(ctrl)

	mockStream.EXPECT().Send(&ethpb.StreamSlotsResponse{Slot: 123}).Do(func(arg0 interface{}) {
		exitRoutine <- true
	})
	mockStream.EXPECT().Context().Return(ctx).AnyTimes()

	go func(tt *testing.T) {
		assert.NoError(tt, server.StreamSlots(&ethpb.StreamSlotsRequest{}, mockStream), "Could not call RPC method")
	}(t)
	wrappedBlk, err := blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 123, Body: &ethpb.BeaconBlockBody{}}})
	require.NoError(t, err)
	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = server.BlockNotifier.BlockFeed().Send(&feed.Event{
			Type: blockfeed.ReceivedBlock,
			Data: &blockfeed.ReceivedBlockData{SignedBlock: wrappedBlk},
		})
	}
	<-exitRoutine
}

func TestServer_StreamSlotsVerified_OnHeadUpdated(t *testing.T) {
	ctx := context.Background()
	chainService := &chainMock.ChainService{}
	server := &Server{
		Ctx:           ctx,
		StateNotifier: chainService.StateNotifier(),
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := mock.NewMockBeaconNodeValidator_StreamSlotsServer(ctrl)
	mockStream.EXPECT().Send(&ethpb.StreamSlotsResponse{Slot: 123}).Do(func(arg0 interface{}) {
		exitRoutine <- true
	})
	mockStream.EXPECT().Context().Return(ctx).AnyTimes()

	go func(tt *testing.T) {
		assert.NoError(tt, server.StreamSlots(&ethpb.StreamSlotsRequest{
			VerifiedOnly: true,
		}, mockStream), "Could not call RPC method")
	}(t)
	wrappedBlk, err := blocks.NewSignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: &ethpb.BeaconBlock{Slot: 123, Body: &ethpb.BeaconBlockBody{}}})
	require.NoError(t, err)
	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = server.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.BlockProcessed,
			Data: &statefeed.BlockProcessedData{Slot: 123, BlockRoot: [32]byte{}, SignedBlock: wrappedBlk},
		})
	}
	<-exitRoutine
}
