// Copyright 2016 The go-ubetprotocol Authors
// This file is part of the go-ubetprotocol library.
//
// The go-ubetprotocol library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ubetprotocol library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ubetprotocol library. If not, see <http://www.gnu.org/licenses/>.

// Package les implements the Light Ubetprotocol Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/ubetprotocol/go-ubetprotocol/accounts"
	"github.com/ubetprotocol/go-ubetprotocol/accounts/abi/bind"
	"github.com/ubetprotocol/go-ubetprotocol/common"
	"github.com/ubetprotocol/go-ubetprotocol/common/hexutil"
	"github.com/ubetprotocol/go-ubetprotocol/common/mclock"
	"github.com/ubetprotocol/go-ubetprotocol/consensus"
	"github.com/ubetprotocol/go-ubetprotocol/core"
	"github.com/ubetprotocol/go-ubetprotocol/core/bloombits"
	"github.com/ubetprotocol/go-ubetprotocol/core/rawdb"
	"github.com/ubetprotocol/go-ubetprotocol/core/types"
	"github.com/ubetprotocol/go-ubetprotocol/eth"
	"github.com/ubetprotocol/go-ubetprotocol/eth/downloader"
	"github.com/ubetprotocol/go-ubetprotocol/eth/filters"
	"github.com/ubetprotocol/go-ubetprotocol/eth/gasprice"
	"github.com/ubetprotocol/go-ubetprotocol/event"
	"github.com/ubetprotocol/go-ubetprotocol/internal/ethapi"
	"github.com/ubetprotocol/go-ubetprotocol/light"
	"github.com/ubetprotocol/go-ubetprotocol/log"
	"github.com/ubetprotocol/go-ubetprotocol/node"
	"github.com/ubetprotocol/go-ubetprotocol/p2p"
	"github.com/ubetprotocol/go-ubetprotocol/p2p/discv5"
	"github.com/ubetprotocol/go-ubetprotocol/params"
	"github.com/ubetprotocol/go-ubetprotocol/rpc"
)

type LightUbetprotocol struct {
	lesCommons

	odr         *LesOdr
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool

	// Handlers
	peers      *peerSet
	txPool     *light.TxPool
	blockchain *light.LightChain
	serverPool *serverPool
	reqDist    *requestDistributor
	retriever  *retrieveManager
	relay      *lesTxRelay

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *eth.Config) (*LightUbetprotocol, error) {
	chainDb, err := ctx.OpenDatabase("lightchaindata", config.DatabaseCache, config.DatabaseHandles, "eth/db/chaindata/")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, config.Genesis, config.ConstantinopleOverride)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	leth := &LightUbetprotocol{
		lesCommons: lesCommons{
			chainDb: chainDb,
			config:  config,
			iConfig: light.DefaultClientIndexerConfig,
		},
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		peers:          peers,
		reqDist:        newRequestDistributor(peers, quitSync, &mclock.System{}),
		accountManager: ctx.AccountManager,
		engine:         eth.CreateConsensusEngine(ctx, chainConfig, &config.Ethash, nil, false, chainDb),
		shutdownChan:   make(chan bool),
		networkId:      config.NetworkId,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   eth.NewBloomIndexer(chainDb, params.BloomBitsBlocksClient, params.HelperTrieConfirmations),
	}

	var trustedNodes []string
	if leth.config.ULC != nil {
		trustedNodes = leth.config.ULC.TrustedServers
	}
	leth.serverPool = newServerPool(chainDb, quitSync, &leth.wg, trustedNodes)
	leth.retriever = newRetrieveManager(peers, leth.reqDist, leth.serverPool)
	leth.relay = newLesTxRelay(peers, leth.retriever)

	leth.odr = NewLesOdr(chainDb, light.DefaultClientIndexerConfig, leth.retriever)
	leth.chtIndexer = light.NewChtIndexer(chainDb, leth.odr, params.CHTFrequency, params.HelperTrieConfirmations)
	leth.bloomTrieIndexer = light.NewBloomTrieIndexer(chainDb, leth.odr, params.BloomBitsBlocksClient, params.BloomTrieFrequency)
	leth.odr.SetIndexers(leth.chtIndexer, leth.bloomTrieIndexer, leth.bloomIndexer)

	checkpoint := config.Checkpoint
	if checkpoint == nil {
		checkpoint = params.TrustedCheckpoints[genesisHash]
	}
	// Note: NewLightChain adds the trusted checkpoint so it needs an ODR with
	// indexers already set but not started yet
	if leth.blockchain, err = light.NewLightChain(leth.odr, leth.chainConfig, leth.engine, checkpoint); err != nil {
		return nil, err
	}
	// Note: AddChildIndexer starts the update process for the child
	leth.bloomIndexer.AddChildIndexer(leth.bloomTrieIndexer)
	leth.chtIndexer.Start(leth.blockchain)
	leth.bloomIndexer.Start(leth.blockchain)

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		leth.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	leth.txPool = light.NewTxPool(leth.chainConfig, leth.blockchain, leth.relay)
	leth.ApiBackend = &LesApiBackend{ctx.ExtRPCEnabled(), leth, nil}

	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.Miner.GasPrice
	}
	leth.ApiBackend.gpo = gasprice.NewOracle(leth.ApiBackend, gpoParams)

	oracle := config.CheckpointOracle
	if oracle == nil {
		oracle = params.CheckpointOracles[genesisHash]
	}
	registrar := newCheckpointOracle(oracle, leth.getLocalCheckpoint)
	if leth.protocolManager, err = NewProtocolManager(leth.chainConfig, checkpoint, light.DefaultClientIndexerConfig, config.ULC, true, config.NetworkId, leth.eventMux, leth.peers, leth.blockchain, nil, chainDb, leth.odr, leth.serverPool, registrar, quitSync, &leth.wg, nil); err != nil {
		return nil, err
	}
	if leth.protocolManager.isULCEnabled() {
		log.Warn("Ultra light client is enabled", "trustedNodes", len(leth.protocolManager.ulc.trustedKeys), "minTrustedFraction", leth.protocolManager.ulc.minTrustedFraction)
		leth.blockchain.DisableCheckFreq()
	}
	return leth, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// Ubetcoinbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Ubetcoinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("mining is not supported in light mode")
}

// Coinbase is the address that mining rewards will be send to (alias for Ubetcoinbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("mining is not supported in light mode")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the ubetprotocol package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightUbetprotocol) APIs() []rpc.API {
	return append(ethapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		}, {
			Namespace: "les",
			Version:   "1.0",
			Service:   NewPrivateLightAPI(&s.lesCommons, s.protocolManager.reg),
			Public:    false,
		},
	}...)
}

func (s *LightUbetprotocol) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightUbetprotocol) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightUbetprotocol) TxPool() *light.TxPool              { return s.txPool }
func (s *LightUbetprotocol) Engine() consensus.Engine           { return s.engine }
func (s *LightUbetprotocol) LesVersion() int                    { return int(ClientProtocolVersions[0]) }
func (s *LightUbetprotocol) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightUbetprotocol) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *LightUbetprotocol) Protocols() []p2p.Protocol {
	return s.makeProtocols(ClientProtocolVersions)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Ubetprotocol protocol implementation.
func (s *LightUbetprotocol) Start(srvr *p2p.Server) error {
	log.Warn("Light client mode is an experimental feature")
	s.startBloomHandlers(params.BloomBitsBlocksClient)
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.networkId)
	// clients are searching for the first advertised protocol in the list
	protocolVersion := AdvertiseProtocolVersions[0]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start(s.config.LightPeers)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Ubetprotocol protocol.
func (s *LightUbetprotocol) Stop() error {
	s.odr.Stop()
	s.relay.Stop()
	s.bloomIndexer.Close()
	s.chtIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()
	s.engine.Close()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}

// SetClient sets the rpc client and binds the registrar contract.
func (s *LightUbetprotocol) SetContractBackend(backend bind.ContractBackend) {
	// Short circuit if registrar is nil
	if s.protocolManager.reg == nil {
		return
	}
	s.protocolManager.reg.start(backend)
}
