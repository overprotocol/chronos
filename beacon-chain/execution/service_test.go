package execution

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/async/event"
	dbutil "github.com/prysmaticlabs/prysm/v5/beacon-chain/db/testing"
	mockExecution "github.com/prysmaticlabs/prysm/v5/beacon-chain/execution/testing"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/execution/types"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/contracts/deposit/mock"
	"github.com/prysmaticlabs/prysm/v5/monitoring/clientstats"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	"github.com/prysmaticlabs/prysm/v5/testing/util"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var _ ChainInfoFetcher = (*Service)(nil)
var _ POWBlockFetcher = (*Service)(nil)
var _ Chain = (*Service)(nil)

type goodLogger struct {
	backend *backends.SimulatedBackend
}

func (_ *goodLogger) Close() {}

func (g *goodLogger) SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error) {
	if g.backend == nil {
		return new(event.Feed).Subscribe(ch), nil
	}
	return g.backend.SubscribeFilterLogs(ctx, q, ch)
}

func (g *goodLogger) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]gethTypes.Log, error) {
	if g.backend == nil {
		logs := make([]gethTypes.Log, 3)
		for i := 0; i < len(logs); i++ {
			logs[i].Address = common.Address{}
			logs[i].Topics = make([]common.Hash, 5)
			logs[i].Topics[0] = common.Hash{'a'}
			logs[i].Topics[1] = common.Hash{'b'}
			logs[i].Topics[2] = common.Hash{'c'}

		}
		return logs, nil
	}
	return g.backend.FilterLogs(ctx, q)
}

type goodNotifier struct {
	MockStateFeed *event.Feed
}

func (g *goodNotifier) StateFeed() event.SubscriberSender {
	if g.MockStateFeed == nil {
		g.MockStateFeed = new(event.Feed)
	}
	return g.MockStateFeed
}

var depositsReqForChainStart = 64

func TestStart_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbutil.SetupDB(t)
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	server, endpoint, err := mockExecution.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoint(endpoint),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup execution service")
	web3Service = setDefaultMocks(web3Service)
	web3Service.rpcClient = &mockExecution.RPCClient{Backend: testAcc.Backend}
	testAcc.Backend.Commit()

	web3Service.Start()
	if len(hook.Entries) > 0 {
		msg := hook.LastEntry().Message
		want := "Could not connect to execution endpoint"
		if strings.Contains(want, msg) {
			t.Errorf("incorrect log, expected %s, got %s", want, msg)
		}
	}
	hook.Reset()
	web3Service.cancel()
}

func TestStart_NoHttpEndpointDefinedFails_WithoutChainStarted(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbutil.SetupDB(t)
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	_, err = NewService(context.Background(),
		WithHttpEndpoint(""),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err)
	require.LogsDoNotContain(t, hook, "missing address")
}

func TestStop_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockExecution.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoint(endpoint),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	web3Service = setDefaultMocks(web3Service)

	testAcc.Backend.Commit()

	err = web3Service.Stop()
	require.NoError(t, err, "Unable to stop web3 ETH1.0 chain service")

	// The context should have been canceled.
	assert.NotNil(t, web3Service.ctx.Err(), "Context wasn't canceled")

	hook.Reset()
}

func TestService_Eth1Synced(t *testing.T) {
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	server, _, err := mockExecution.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})

	currTime := testAcc.Backend.Blockchain().CurrentHeader().Time
	now := time.Now()
	assert.NoError(t, testAcc.Backend.AdjustTime(now.Sub(time.Unix(int64(currTime), 0))))
	testAcc.Backend.Commit()
}

func TestFollowBlock_OK(t *testing.T) {
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockExecution.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoint(endpoint),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	// simulated backend sets eth1 block
	// time as 10 seconds
	params.SetupTestConfigCleanup(t)
	conf := params.BeaconConfig().Copy()
	conf.SecondsPerETH1Block = 10
	params.OverrideBeaconConfig(conf)

	web3Service = setDefaultMocks(web3Service)
	web3Service.rpcClient = &mockExecution.RPCClient{Backend: testAcc.Backend}
	baseHeight := testAcc.Backend.Blockchain().CurrentBlock().Number.Uint64()
	// process follow_distance blocks
	for i := 0; i < int(params.BeaconConfig().Eth1FollowDistance); i++ {
		testAcc.Backend.Commit()
	}
	// set current height
	web3Service.latestEth1Data.BlockHeight = testAcc.Backend.Blockchain().CurrentBlock().Number.Uint64()
	web3Service.latestEth1Data.BlockTime = testAcc.Backend.Blockchain().CurrentBlock().Time

	h, err := web3Service.followedBlockHeight(context.Background())
	require.NoError(t, err)
	assert.Equal(t, baseHeight, h, "Unexpected block height")
	numToForward := uint64(2)
	expectedHeight := numToForward + baseHeight
	// forward 2 blocks
	for i := uint64(0); i < numToForward; i++ {
		testAcc.Backend.Commit()
	}
	// set current height
	web3Service.latestEth1Data.BlockHeight = testAcc.Backend.Blockchain().CurrentBlock().Number.Uint64()
	web3Service.latestEth1Data.BlockTime = testAcc.Backend.Blockchain().CurrentBlock().Time

	h, err = web3Service.followedBlockHeight(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedHeight, h, "Unexpected block height")
}

func TestStatus(t *testing.T) {
	now := time.Now()

	beforeFiveMinutesAgo := uint64(now.Add(-5*time.Minute - 30*time.Second).Unix())
	afterFiveMinutesAgo := uint64(now.Add(-5*time.Minute + 30*time.Second).Unix())

	testCases := map[*Service]string{
		// "status is ok" cases
		{}: "",
		{isRunning: true, latestEth1Data: &ethpb.LatestETH1Data{BlockTime: afterFiveMinutesAgo}}:   "",
		{isRunning: false, latestEth1Data: &ethpb.LatestETH1Data{BlockTime: beforeFiveMinutesAgo}}: "",
		{isRunning: false, runError: errors.New("test runError")}:                                  "",
		// "status is error" cases
		{isRunning: true, runError: errors.New("test runError")}: "test runError",
	}

	for web3ServiceState, wantedErrorText := range testCases {
		status := web3ServiceState.Status()
		if status == nil {
			assert.Equal(t, "", wantedErrorText)

		} else {
			assert.Equal(t, wantedErrorText, status.Error())
		}
	}
}

func TestHandlePanic_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockExecution.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoint(endpoint),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	// nil eth1DataFetcher would panic if cached value not used
	web3Service.rpcClient = nil
	web3Service.processBlockHeader(nil)
	require.LogsContain(t, hook, "Panicked when handling data from ETH 1.0 Chain!")
}

func TestLogTillGenesis_OK(t *testing.T) {
	// Reset the var at the end of the test.
	currPeriod := logPeriod
	logPeriod = 1 * time.Second
	defer func() {
		logPeriod = currPeriod
	}()

	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig().Copy()
	cfg.Eth1FollowDistance = 5
	params.OverrideBeaconConfig(cfg)

	nCfg := params.BeaconNetworkConfig()
	nCfg.ContractDeploymentBlock = 0
	params.OverrideBeaconNetworkConfig(nCfg)

	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockExecution.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoint(endpoint),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")

	web3Service.rpcClient = &mockExecution.RPCClient{Backend: testAcc.Backend}
	web3Service.httpLogger = testAcc.Backend
	for i := 0; i < 30; i++ {
		testAcc.Backend.Commit()
	}
	web3Service.latestEth1Data = &ethpb.LatestETH1Data{LastRequestedBlock: 0}
	// Spin off to a separate routine
	go web3Service.run(web3Service.ctx.Done())
	// Wait for 2 seconds so that the
	// info is logged.
	time.Sleep(2 * time.Second)
	web3Service.cancel()
}

func TestNewService_EarliestVotingBlock(t *testing.T) {
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)
	server, endpoint, err := mockExecution.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	web3Service, err := NewService(context.Background(),
		WithHttpEndpoint(endpoint),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	// simulated backend sets eth1 block
	// time as 10 seconds
	params.SetupTestConfigCleanup(t)
	conf := params.BeaconConfig().Copy()
	conf.SecondsPerETH1Block = 10
	conf.Eth1FollowDistance = 50
	params.OverrideBeaconConfig(conf)

	numToForward := 1500
	// forward 1500 blocks
	for i := 0; i < numToForward; i++ {
		testAcc.Backend.Commit()
	}
	currTime := testAcc.Backend.Blockchain().CurrentHeader().Time
	now := time.Now()
	err = testAcc.Backend.AdjustTime(now.Sub(time.Unix(int64(currTime), 0)))
	require.NoError(t, err)
	testAcc.Backend.Commit()

	web3Service.latestEth1Data.BlockHeight = testAcc.Backend.Blockchain().CurrentHeader().Number.Uint64()
	web3Service.latestEth1Data.BlockTime = testAcc.Backend.Blockchain().CurrentHeader().Time
}

func TestNewService_Eth1HeaderRequLimit(t *testing.T) {
	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)

	server, endpoint, err := mockExecution.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	s1, err := NewService(context.Background(),
		WithHttpEndpoint(endpoint),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	assert.Equal(t, defaultEth1HeaderReqLimit, s1.cfg.eth1HeaderReqLimit, "default eth1 header request limit not set")
	s2, err := NewService(context.Background(),
		WithHttpEndpoint(endpoint),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
		WithEth1HeaderRequestLimit(uint64(150)),
	)
	require.NoError(t, err, "unable to setup web3 ETH1.0 chain service")
	assert.Equal(t, uint64(150), s2.cfg.eth1HeaderReqLimit, "unable to set eth1HeaderRequestLimit")
}

type mockBSUpdater struct {
	lastBS clientstats.BeaconNodeStats
}

func (mbs *mockBSUpdater) Update(bs clientstats.BeaconNodeStats) {
	mbs.lastBS = bs
}

var _ BeaconNodeStatsUpdater = &mockBSUpdater{}

func TestDedupEndpoints(t *testing.T) {
	assert.DeepEqual(t, []string{"A"}, dedupEndpoints([]string{"A"}), "did not dedup correctly")
	assert.DeepEqual(t, []string{"A", "B"}, dedupEndpoints([]string{"A", "B"}), "did not dedup correctly")
	assert.DeepEqual(t, []string{"A", "B"}, dedupEndpoints([]string{"A", "A", "A", "B"}), "did not dedup correctly")
	assert.DeepEqual(t, []string{"A", "B"}, dedupEndpoints([]string{"A", "A", "A", "B", "B"}), "did not dedup correctly")
}

func Test_batchRequestHeaders_UnderflowChecks(t *testing.T) {
	srv := &Service{}
	start := uint64(101)
	end := uint64(100)
	_, err := srv.batchRequestHeaders(start, end)
	require.ErrorContains(t, "cannot be >", err)

	start = uint64(200)
	end = uint64(100)
	_, err = srv.batchRequestHeaders(start, end)
	require.ErrorContains(t, "cannot be >", err)
}

func TestService_EnsureConsistentPowchainData(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	srv, endpoint, err := mockExecution.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		srv.Stop()
	})
	s1, err := NewService(context.Background(),
		WithHttpEndpoint(endpoint),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err)
	genState, err := util.NewBeaconState()
	require.NoError(t, err)
	assert.NoError(t, genState.SetSlot(1000))

	require.NoError(t, s1.cfg.beaconDB.SaveGenesisData(context.Background(), genState))
	_, err = s1.validPowchainData(context.Background())
	require.NoError(t, err)

	eth1Data, err := s1.cfg.beaconDB.ExecutionChainData(context.Background())
	assert.NoError(t, err)

	assert.NotNil(t, eth1Data)
}

func TestService_InitializeCorrectly(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	srv, endpoint, err := mockExecution.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		srv.Stop()
	})
	s1, err := NewService(context.Background(),
		WithHttpEndpoint(endpoint),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err)
	genState, err := util.NewBeaconState()
	require.NoError(t, err)
	assert.NoError(t, genState.SetSlot(1000))

	require.NoError(t, s1.cfg.beaconDB.SaveGenesisData(context.Background(), genState))
	_, err = s1.validPowchainData(context.Background())
	require.NoError(t, err)

	eth1Data, err := s1.cfg.beaconDB.ExecutionChainData(context.Background())
	assert.NoError(t, err)

	assert.NoError(t, s1.initializeEth1Data(context.Background(), eth1Data))
	assert.Equal(t, int64(-1), s1.lastReceivedMerkleIndex, "received incorrect last received merkle index")
}

func TestService_EnsureValidPowchainData(t *testing.T) {
	beaconDB := dbutil.SetupDB(t)
	srv, endpoint, err := mockExecution.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		srv.Stop()
	})
	s1, err := NewService(context.Background(),
		WithHttpEndpoint(endpoint),
		WithDatabase(beaconDB),
	)
	require.NoError(t, err)
	genState, err := util.NewBeaconState()
	require.NoError(t, err)
	assert.NoError(t, genState.SetSlot(1000))

	require.NoError(t, s1.cfg.beaconDB.SaveGenesisData(context.Background(), genState))

	_, err = s1.validPowchainData(context.Background())
	require.NoError(t, err)

	eth1Data, err := s1.cfg.beaconDB.ExecutionChainData(context.Background())
	assert.NoError(t, err)

	assert.NotNil(t, eth1Data)
}

func TestETH1Endpoints(t *testing.T) {
	server, firstEndpoint, err := mockExecution.SetupRPCServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		server.Stop()
	})
	endpoints := []string{firstEndpoint}

	testAcc, err := mock.Setup()
	require.NoError(t, err, "Unable to set up simulated backend")
	beaconDB := dbutil.SetupDB(t)

	mbs := &mockBSUpdater{}
	s1, err := NewService(context.Background(),
		WithHttpEndpoint(endpoints[0]),
		WithDepositContractAddress(testAcc.ContractAddr),
		WithDatabase(beaconDB),
		WithBeaconNodeStatsUpdater(mbs),
	)
	s1.cfg.beaconNodeStatsUpdater = mbs
	require.NoError(t, err)

	// Check default endpoint is set to current.
	assert.Equal(t, firstEndpoint, s1.ExecutionClientEndpoint(), "Unexpected http endpoint")
}

func TestService_CacheBlockHeaders(t *testing.T) {
	rClient := &slowRPCClient{limit: 1000}
	s := &Service{
		cfg:         &config{eth1HeaderReqLimit: 1000},
		rpcClient:   rClient,
		headerCache: newHeaderCache(),
	}
	assert.NoError(t, s.cacheBlockHeaders(1, 1000))
	assert.Equal(t, 1, rClient.numOfCalls)
	// Reset Num of Calls
	rClient.numOfCalls = 0
	// Increase header request limit to trigger the batch limiting
	// code path.
	s.cfg.eth1HeaderReqLimit = 1001

	assert.NoError(t, s.cacheBlockHeaders(1000, 3000))
	// 1000 - 2000 would be 1001 headers which is higher than our request limit, it
	// is then reduced to 500 and tried again.
	assert.Equal(t, 5, rClient.numOfCalls)
}

func TestService_FollowBlock(t *testing.T) {
	followTime := params.BeaconConfig().Eth1FollowDistance * params.BeaconConfig().SecondsPerETH1Block
	followTime += 10000
	bMap := make(map[uint64]*types.HeaderInfo)
	for i := uint64(3000); i > 0; i-- {
		h := &gethTypes.Header{
			Number: big.NewInt(int64(i)),
			Time:   followTime + (i * 40),
		}
		bMap[i] = &types.HeaderInfo{
			Number: h.Number,
			Hash:   h.Hash(),
			Time:   h.Time,
		}
	}
	s := &Service{
		cfg:            &config{eth1HeaderReqLimit: 1000},
		rpcClient:      &mockExecution.RPCClient{BlockNumMap: bMap},
		headerCache:    newHeaderCache(),
		latestEth1Data: &ethpb.LatestETH1Data{BlockTime: (3000 * 40) + followTime, BlockHeight: 3000},
	}
	h, err := s.followedBlockHeight(context.Background())
	assert.NoError(t, err)
	// With a much higher blocktime, the follow height is respectively shortened.
	assert.Equal(t, uint64(2692), h)
}

type slowRPCClient struct {
	limit      int
	numOfCalls int
}

func (s *slowRPCClient) Close() {
	panic("implement me")
}

func (s *slowRPCClient) BatchCall(b []rpc.BatchElem) error {
	s.numOfCalls++
	if len(b) > s.limit {
		return errTimedOut
	}
	for _, e := range b {
		num, err := hexutil.DecodeBig(e.Args[0].(string))
		if err != nil {
			return err
		}
		h := &gethTypes.Header{Number: num}
		*e.Result.(*types.HeaderInfo) = types.HeaderInfo{Number: h.Number, Hash: h.Hash()}
	}
	return nil
}

func (s *slowRPCClient) CallContext(_ context.Context, _ interface{}, _ string, _ ...interface{}) error {
	panic("implement me")
}
