// Package execution defines a runtime service which is tasked with
// communicating with an eth1 endpoint, processing logs from a deposit
// contract, and the latest eth1 data headers for usage in the beacon node.
package execution

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethRPC "github.com/ethereum/go-ethereum/rpc"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	statefeed "github.com/prysmaticlabs/prysm/v5/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/execution/types"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state"
	native "github.com/prysmaticlabs/prysm/v5/beacon-chain/state/state-native"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/monitoring/clientstats"
	"github.com/prysmaticlabs/prysm/v5/network"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
)

var (
	validDepositsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powchain_valid_deposits_received",
		Help: "The number of valid deposits received in the deposit contract",
	})
	blockNumberGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powchain_block_number",
		Help: "The current block number in the proof-of-work chain",
	})
	missedDepositLogsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "powchain_missed_deposit_logs",
		Help: "The number of times a missed deposit log is detected",
	})
)

var (
	// time to wait before trying to reconnect with the eth1 node.
	backOffPeriod = 15 * time.Second
	// amount of times before we log the status of the eth1 dial attempt.
	logThreshold = 8
	// period to log chainstart related information
	logPeriod = 1 * time.Minute
)

// ChainInfoFetcher retrieves information about eth1 metadata at the Ethereum consensus genesis time.
type ChainInfoFetcher interface {
	ExecutionClientConnected() bool
	ExecutionClientEndpoint() string
	ExecutionClientConnectionErr() error
}

// POWBlockFetcher defines a struct that can retrieve mainchain blocks.
type POWBlockFetcher interface {
	BlockTimeByHeight(ctx context.Context, height *big.Int) (uint64, error)
	BlockByTimestamp(ctx context.Context, time uint64) (*types.HeaderInfo, error)
	BlockHashByHeight(ctx context.Context, height *big.Int) (common.Hash, error)
	BlockExists(ctx context.Context, hash common.Hash) (bool, *big.Int, error)
}

// Chain defines a standard interface for the powchain service in Prysm.
type Chain interface {
	ChainInfoFetcher
	POWBlockFetcher
}

// RPCClient defines the rpc methods required to interact with the eth1 node.
type RPCClient interface {
	Close()
	BatchCall(b []gethRPC.BatchElem) error
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

type RPCClientEmpty struct {
}

func (RPCClientEmpty) Close() {}
func (RPCClientEmpty) BatchCall([]gethRPC.BatchElem) error {
	return errors.New("rpc client is not initialized")
}

func (RPCClientEmpty) CallContext(context.Context, interface{}, string, ...interface{}) error {
	return errors.New("rpc client is not initialized")
}

// config defines a config struct for dependencies into the service.
type config struct {
	depositContractAddr     common.Address
	beaconDB                db.HeadAccessDatabase
	stateNotifier           statefeed.Notifier
	stateGen                *stategen.State
	eth1HeaderReqLimit      uint64
	beaconNodeStatsUpdater  BeaconNodeStatsUpdater
	currHttpEndpoint        network.Endpoint
	headers                 []string
	finalizedStateAtStartup state.BeaconState
	jwtId                   string
}

// Service fetches important information about the canonical
// eth1 chain via a web3 endpoint using an ethclient.
// The beacon chain requires synchronization with the eth1 chain's current
// block hash, block number, and access to logs within the
// Validator Registration Contract on the eth1 chain to kick off the beacon
// chain's validator registration process.
type Service struct {
	connectedETH1           bool
	isRunning               bool
	processingLock          sync.RWMutex
	latestEth1DataLock      sync.RWMutex
	cfg                     *config
	ctx                     context.Context
	cancel                  context.CancelFunc
	eth1HeadTicker          *time.Ticker
	httpLogger              bind.ContractFilterer
	rpcClient               RPCClient
	headerCache             *headerCache // cache to store block hash/block height.
	latestEth1Data          *ethpb.LatestETH1Data
	lastReceivedMerkleIndex int64 // Keeps track of the last received index to prevent log spam.
	runError                error
	preGenesisState         state.BeaconState
}

// NewService sets up a new instance with an ethclient when given a web3 endpoint as a string in the config.
func NewService(ctx context.Context, opts ...Option) (*Service, error) {
	ctx, cancel := context.WithCancel(ctx)
	_ = cancel // govet fix for lost cancel. Cancel is handled in service.Stop()
	var err error
	genState, err := transition.EmptyGenesisState()
	if err != nil {
		return nil, errors.Wrap(err, "could not set up genesis state")
	}

	s := &Service{
		ctx:       ctx,
		cancel:    cancel,
		rpcClient: RPCClientEmpty{},
		cfg: &config{
			beaconNodeStatsUpdater: &NopBeaconNodeStatsUpdater{},
			eth1HeaderReqLimit:     defaultEth1HeaderReqLimit,
		},
		latestEth1Data: &ethpb.LatestETH1Data{
			BlockHeight:        0,
			BlockTime:          0,
			BlockHash:          []byte{},
			LastRequestedBlock: 0,
		},
		headerCache:             newHeaderCache(),
		lastReceivedMerkleIndex: -1,
		preGenesisState:         genState,
		eth1HeadTicker:          time.NewTicker(time.Duration(params.BeaconConfig().SecondsPerETH1Block) * time.Second),
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	eth1Data, err := s.validPowchainData(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to validate powchain data")
	}
	if err := s.initializeEth1Data(ctx, eth1Data); err != nil {
		return nil, err
	}
	return s, nil
}

// Start the powchain service's main event loop.
func (s *Service) Start() {
	if err := s.setupExecutionClientConnections(s.ctx, s.cfg.currHttpEndpoint); err != nil {
		log.WithError(err).Error("Could not connect to execution endpoint")
	}
	// If the chain has not started already and we don't have access to eth1 nodes, we will not be
	// able to generate the genesis state.
	if /*!s.chainStartData.Chainstarted &&*/ s.cfg.currHttpEndpoint.Url == "" {
		// check for genesis state before shutting down the node,
		// if a genesis state exists, we can continue on.
		genState, err := s.cfg.beaconDB.GenesisState(s.ctx)
		if err != nil {
			log.Fatal(err)
		}
		if genState == nil || genState.IsNil() {
			log.Fatal("no eth1 http endpoint & genesis state defined")
		}
	}

	s.isRunning = true

	// Poll the execution client connection and fallback if errors occur.
	s.pollConnectionStatus(s.ctx)

	go s.run(s.ctx.Done())
}

// Stop the web3 service's main event loop and associated goroutines.
func (s *Service) Stop() error {
	if s.cancel != nil {
		defer s.cancel()
	}
	if s.rpcClient != nil {
		s.rpcClient.Close()
	}
	return nil
}

// Status is service health checks. Return nil or error.
func (s *Service) Status() error {
	// Service don't start
	if !s.isRunning {
		return nil
	}
	// get error from run function
	return s.runError
}

// ExecutionClientConnected checks whether are connected via RPC.
func (s *Service) ExecutionClientConnected() bool {
	return s.connectedETH1
}

// ExecutionClientEndpoint returns the URL of the current, connected execution client.
func (s *Service) ExecutionClientEndpoint() string {
	return s.cfg.currHttpEndpoint.Url
}

// ExecutionClientConnectionErr returns the error (if any) of the current connection.
func (s *Service) ExecutionClientConnectionErr() error {
	return s.runError
}

func (s *Service) updateBeaconNodeStats() {
	bs := clientstats.BeaconNodeStats{}
	if s.ExecutionClientConnected() {
		bs.SyncEth1Connected = true
	}
	s.cfg.beaconNodeStatsUpdater.Update(bs)
}

func (s *Service) updateConnectedETH1(state bool) {
	s.connectedETH1 = state
	s.updateBeaconNodeStats()
}

// refers to the latest eth1 block which follows the condition: eth1_timestamp +
// SECONDS_PER_ETH1_BLOCK * ETH1_FOLLOW_DISTANCE <= current_unix_time
func (s *Service) followedBlockHeight(ctx context.Context) (uint64, error) {
	followTime := params.BeaconConfig().Eth1FollowDistance * params.BeaconConfig().SecondsPerETH1Block
	latestBlockTime := uint64(0)
	if s.latestEth1Data.BlockTime > followTime {
		latestBlockTime = s.latestEth1Data.BlockTime - followTime
		// This should only come into play in testnets - when the chain hasn't advanced past the follow distance,
		// we don't want to consider any block before the genesis block.
		if s.latestEth1Data.BlockHeight < params.BeaconConfig().Eth1FollowDistance {
			latestBlockTime = s.latestEth1Data.BlockTime
		}
	}
	blk, err := s.BlockByTimestamp(ctx, latestBlockTime)
	if err != nil {
		return 0, errors.Wrapf(err, "BlockByTimestamp=%d", latestBlockTime)
	}
	return blk.Number.Uint64(), nil
}

// processBlockHeader adds a newly observed eth1 block to the block cache and
// updates the latest blockHeight, blockHash, and blockTime properties of the service.
func (s *Service) processBlockHeader(header *types.HeaderInfo) {
	defer safelyHandlePanic()
	blockNumberGauge.Set(float64(header.Number.Int64()))
	s.latestEth1DataLock.Lock()
	s.latestEth1Data.BlockHeight = header.Number.Uint64()
	s.latestEth1Data.BlockHash = header.Hash.Bytes()
	s.latestEth1Data.BlockTime = header.Time
	s.latestEth1DataLock.Unlock()
	log.WithFields(logrus.Fields{
		"blockNumber": s.latestEth1Data.BlockHeight,
		"blockHash":   hexutil.Encode(s.latestEth1Data.BlockHash),
	}).Debug("Latest eth1 chain event")
}

// batchRequestHeaders requests the block range specified in the arguments. Instead of requesting
// each block in one call, it batches all requests into a single rpc call.
func (s *Service) batchRequestHeaders(startBlock, endBlock uint64) ([]*types.HeaderInfo, error) {
	if startBlock > endBlock {
		return nil, fmt.Errorf("start block height %d cannot be > end block height %d", startBlock, endBlock)
	}
	requestRange := (endBlock - startBlock) + 1
	elems := make([]gethRPC.BatchElem, 0, requestRange)
	headers := make([]*types.HeaderInfo, 0, requestRange)
	for i := startBlock; i <= endBlock; i++ {
		header := &types.HeaderInfo{}
		elems = append(elems, gethRPC.BatchElem{
			Method: "eth_getBlockByNumber",
			Args:   []interface{}{hexutil.EncodeBig(new(big.Int).SetUint64(i)), false},
			Result: header,
			Error:  error(nil),
		})
		headers = append(headers, header)
	}
	ioErr := s.rpcClient.BatchCall(elems)
	if ioErr != nil {
		return nil, ioErr
	}
	for _, e := range elems {
		if e.Error != nil {
			return nil, e.Error
		}
	}
	for _, h := range headers {
		if h != nil {
			if err := s.headerCache.AddHeader(h); err != nil {
				return nil, err
			}
		}
	}
	return headers, nil
}

// safelyHandleHeader will recover and log any panic that occurs from the block
func safelyHandlePanic() {
	if r := recover(); r != nil {
		log.WithFields(logrus.Fields{
			"r": r,
		}).Error("Panicked when handling data from ETH 1.0 Chain! Recovering...")

		debug.PrintStack()
	}
}

func (s *Service) initPOWService() {
	// Use a custom logger to only log errors
	logCounter := 0
	errorLogger := func(err error, msg string) {
		if logCounter > logThreshold {
			log.WithError(err).Error(msg)
			logCounter = 0
		}
		logCounter++
	}

	// Run in a select loop to retry in the event of any failures.
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			ctx := s.ctx
			header, err := s.HeaderByNumber(ctx, nil)
			if err != nil {
				err = errors.Wrap(err, "HeaderByNumber")
				s.retryExecutionClientConnection(ctx, err)
				errorLogger(err, "Unable to retrieve latest execution client header")
				continue
			}

			s.latestEth1DataLock.Lock()
			s.latestEth1Data.BlockHeight = header.Number.Uint64()
			s.latestEth1Data.BlockHash = header.Hash.Bytes()
			s.latestEth1Data.BlockTime = header.Time
			s.latestEth1DataLock.Unlock()
			return
		}
	}
}

// run subscribes to all the services for the eth1 chain.
func (s *Service) run(done <-chan struct{}) {
	s.runError = nil

	// Do not keep storing the finalized state as it is
	// no longer of use.
	s.removeStartupState()

	for {
		select {
		case <-done:
			s.isRunning = false
			s.runError = nil
			s.rpcClient.Close()
			s.updateConnectedETH1(false)
			log.Debug("Context closed, exiting goroutine")
			return
		case <-s.eth1HeadTicker.C:
			head, err := s.HeaderByNumber(s.ctx, nil)
			if err != nil {
				s.pollConnectionStatus(s.ctx)
				log.WithError(err).Debug("Could not fetch latest eth1 header")
				continue
			}
			s.processBlockHeader(head)
		}
	}
}

// Caches block headers from the desired range.
func (s *Service) cacheBlockHeaders(start, end uint64) error {
	batchSize := s.cfg.eth1HeaderReqLimit
	for i := start; i < end; i += batchSize {
		startReq := i
		endReq := i + batchSize
		if endReq > 0 {
			// Reduce the end request by one
			// to prevent total batch size from exceeding
			// the allotted limit.
			endReq -= 1
		}
		endReq = min(endReq, end)
		// We call batchRequestHeaders for its header caching side-effect, so we don't need the return value.
		_, err := s.batchRequestHeaders(startReq, endReq)
		if err != nil {
			if clientTimedOutError(err) {
				// Reduce batch size as eth1 node is
				// unable to respond to the request in time.
				batchSize /= 2
				// Always have it greater than 0.
				if batchSize == 0 {
					batchSize += 1
				}

				// Reset request value
				if i > batchSize {
					i -= batchSize
				}
				continue
			}
			return errors.Wrapf(err, "cacheBlockHeaders, start=%d, end=%d", startReq, endReq)
		}
	}
	return nil
}

// initializes our service from the provided eth1data object by initializing all the relevant
// fields and data.
func (s *Service) initializeEth1Data(ctx context.Context, eth1DataInDB *ethpb.ETH1ChainData) error {
	// The node has no eth1data persisted on disk, so we exit and instead
	// request from contract logs.
	if eth1DataInDB == nil {
		return nil
	}
	var err error
	if !reflect.ValueOf(eth1DataInDB.BeaconState).IsZero() {
		s.preGenesisState, err = native.InitializeFromProtoPhase0(eth1DataInDB.BeaconState)
		if err != nil {
			return errors.Wrap(err, "Could not initialize state trie")
		}
	}
	s.latestEth1Data = eth1DataInDB.CurrentEth1Data

	return nil
}

// Validates the current powchain data is saved and makes sure that any
// embedded genesis state is correctly accounted for.
func (s *Service) validPowchainData(ctx context.Context) (*ethpb.ETH1ChainData, error) {
	genState, err := s.cfg.beaconDB.GenesisState(ctx)
	if err != nil {
		return nil, err
	}
	eth1Data, err := s.cfg.beaconDB.ExecutionChainData(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to retrieve eth1 data")
	}
	if genState == nil || genState.IsNil() {
		return eth1Data, nil
	}
	if eth1Data == nil {
		pbState, err := native.ProtobufBeaconStatePhase0(s.preGenesisState.ToProtoUnsafe())
		if err != nil {
			return nil, err
		}

		eth1Data = &ethpb.ETH1ChainData{
			CurrentEth1Data: s.latestEth1Data,
			BeaconState:     pbState,
		}
		if err := s.cfg.beaconDB.SaveExecutionChainData(ctx, eth1Data); err != nil {
			return nil, err
		}
	}
	return eth1Data, nil
}

func dedupEndpoints(endpoints []string) []string {
	selectionMap := make(map[string]bool)
	newEndpoints := make([]string, 0, len(endpoints))
	for _, point := range endpoints {
		if selectionMap[point] {
			continue
		}
		newEndpoints = append(newEndpoints, point)
		selectionMap[point] = true
	}
	return newEndpoints
}

func (s *Service) removeStartupState() {
	s.cfg.finalizedStateAtStartup = nil
}
