// Copyright (c) 2017-2021 420Integrated Devlopment Team
// This file is part of the go-highcoin library.
//
// The go-highcoin library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-highcoin library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-highcoin library. If not, see <http://www.gnu.org/licenses/>.

// Package les implements the Light Highcoin Subprotocol.
package les

import (
	"fmt"
	"time"

	"github.com/420integrated/go-highcoin/accounts"
	"github.com/420integrated/go-highcoin/common"
	"github.com/420integrated/go-highcoin/common/hexutil"
	"github.com/420integrated/go-highcoin/common/mclock"
	"github.com/420integrated/go-highcoin/consensus"
	"github.com/420integrated/go-highcoin/core"
	"github.com/420integrated/go-highcoin/core/bloombits"
	"github.com/420integrated/go-highcoin/core/rawdb"
	"github.com/420integrated/go-highcoin/core/types"
	"github.com/420integrated/go-highcoin/high/downloader"
	"github.com/420integrated/go-highcoin/high/highconfig"
	"github.com/420integrated/go-highcoin/high/filters"
	"github.com/420integrated/go-highcoin/high/smokeprice"
	"github.com/420integrated/go-highcoin/event"
	"github.com/420integrated/go-highcoin/internal/highapi"
	vfc "github.com/420integrated/go-highcoin/les/vflux/client"
	"github.com/420integrated/go-highcoin/light"
	"github.com/420integrated/go-highcoin/log"
	"github.com/420integrated/go-highcoin/node"
	"github.com/420integrated/go-highcoin/p2p"
	"github.com/420integrated/go-highcoin/p2p/enode"
	"github.com/420integrated/go-highcoin/params"
	"github.com/420integrated/go-highcoin/rpc"
)

type LightHighcoin struct {
	lesCommons

	peers          *serverPeerSet
	reqDist        *requestDistributor
	retriever      *retrieveManager
	odr            *LesOdr
	relay          *lesTxRelay
	handler        *clientHandler
	txPool         *light.TxPool
	blockchain     *light.LightChain
	serverPool     *vfc.ServerPool
	dialCandidates enode.Iterator
	pruner         *pruner

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend     *LesApiBackend
	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager
	netRPCService  *highapi.PublicNetAPI

	p2pServer *p2p.Server
	p2pConfig *p2p.Config
}

// New creates an instance of the light client.
func New(stack *node.Node, config *highconfig.Config) (*LightHighcoin, error) {
	chainDb, err := stack.OpenDatabase("lightchaindata", config.DatabaseCache, config.DatabaseHandles, "high/db/chaindata/")
	if err != nil {
		return nil, err
	}
	lesDb, err := stack.OpenDatabase("les.client", 0, 0, "high/db/les.client")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newServerPeerSet()
	lhigh := &LightHighcoin{
		lesCommons: lesCommons{
			genesis:     genesisHash,
			config:      config,
			chainConfig: chainConfig,
			iConfig:     light.DefaultClientIndexerConfig,
			chainDb:     chainDb,
			lesDb:       lesDb,
			closeCh:     make(chan struct{}),
		},
		peers:          peers,
		eventMux:       stack.EventMux(),
		reqDist:        newRequestDistributor(peers, &mclock.System{}),
		accountManager: stack.AccountManager(),
		engine:         highconfig.CreateConsensusEngine(stack, chainConfig, &config.Ethash, nil, false, chainDb),
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   core.NewBloomIndexer(chainDb, params.BloomBitsBlocksClient, params.HelperTrieConfirmations),
		p2pServer:      stack.Server(),
		p2pConfig:      &stack.Config().P2P,
	}

	lhigh.serverPool, lhigh.dialCandidates = vfc.NewServerPool(lesDb, []byte("serverpool:"), time.Second, nil, &mclock.System{}, config.UltraLightServers, requestList)
	lhigh.serverPool.AddMetrics(suggestedTimeoutGauge, totalValueGauge, serverSelectableGauge, serverConnectedGauge, sessionValueMeter, serverDialedMeter)

	lhigh.retriever = newRetrieveManager(peers, lhigh.reqDist, lhigh.serverPool.GetTimeout)
	lhigh.relay = newLesTxRelay(peers, lhigh.retriever)

	lhigh.odr = NewLesOdr(chainDb, light.DefaultClientIndexerConfig, lhigh.peers, lhigh.retriever)
	lhigh.chtIndexer = light.NewChtIndexer(chainDb, lhigh.odr, params.CHTFrequency, params.HelperTrieConfirmations, config.LightNoPrune)
	lhigh.bloomTrieIndexer = light.NewBloomTrieIndexer(chainDb, lhigh.odr, params.BloomBitsBlocksClient, params.BloomTrieFrequency, config.LightNoPrune)
	lhigh.odr.SetIndexers(lhigh.chtIndexer, lhigh.bloomTrieIndexer, lhigh.bloomIndexer)

	checkpoint := config.Checkpoint
	if checkpoint == nil {
		checkpoint = params.TrustedCheckpoints[genesisHash]
	}
	// Note: NewLightChain adds the trusted checkpoint so it needs an ODR with
	// indexers already set but not started yet
	if lhigh.blockchain, err = light.NewLightChain(lhigh.odr, lhigh.chainConfig, lhigh.engine, checkpoint); err != nil {
		return nil, err
	}
	lhigh.chainReader = lhigh.blockchain
	lhigh.txPool = light.NewTxPool(lhigh.chainConfig, lhigh.blockchain, lhigh.relay)

	// Set up checkpoint oracle.
	lhigh.oracle = lhigh.setupOracle(stack, genesisHash, config)

	// Note: AddChildIndexer starts the update process for the child
	lhigh.bloomIndexer.AddChildIndexer(lhigh.bloomTrieIndexer)
	lhigh.chtIndexer.Start(lhigh.blockchain)
	lhigh.bloomIndexer.Start(lhigh.blockchain)

	// Start a light chain pruner to delete useless historical data.
	lhigh.pruner = newPruner(chainDb, lhigh.chtIndexer, lhigh.bloomTrieIndexer)

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lhigh.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lhigh.ApiBackend = &LesApiBackend{stack.Config().ExtRPCEnabled(), stack.Config().AllowUnprotectedTxs, lhigh, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.Miner.SmokePrice
	}
	lhigh.ApiBackend.gpo = smokeprice.NewOracle(lhigh.ApiBackend, gpoParams)

	lhigh.handler = newClientHandler(config.UltraLightServers, config.UltraLightFraction, checkpoint, lhigh)
	if lhigh.handler.ulc != nil {
		log.Warn("Ultra light client is enabled", "trustedNodes", len(lhigh.handler.ulc.keys), "minTrustedFraction", lhigh.handler.ulc.fraction)
		lhigh.blockchain.DisableCheckFreq()
	}

	lhigh.netRPCService = highapi.NewPublicNetAPI(lhigh.p2pServer, lhigh.config.NetworkId)

	// Register the backend on the node
	stack.RegisterAPIs(lhigh.APIs())
	stack.RegisterProtocols(lhigh.Protocols())
	stack.RegisterLifecycle(lhigh)

	// Check for unclean shutdown
	if uncleanShutdowns, discards, err := rawdb.PushUncleanShutdownMarker(chainDb); err != nil {
		log.Error("Could not update unclean-shutdown-marker list", "error", err)
	} else {
		if discards > 0 {
			log.Warn("Old unclean shutdowns found", "count", discards)
		}
		for _, tstamp := range uncleanShutdowns {
			t := time.Unix(int64(tstamp), 0)
			log.Warn("Unclean shutdown detected", "booted", t,
				"age", common.PrettyAge(t))
		}
	}
	return lhigh, nil
}

type LightDummyAPI struct{}

// Highcoinbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Highcoinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("mining is not supported in light mode")
}

// Coinbase is the address that mining rewards will be send to (alias for Highcoinbase)
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

// APIs returns the collection of RPC services the highcoin package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightHighcoin) APIs() []rpc.API {
	apis := highapi.GetAPIs(s.ApiBackend)
	apis = append(apis, s.engine.APIs(s.BlockChain().HeaderChain())...)
	return append(apis, []rpc.API{
		{
			Namespace: "high",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "high",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.handler.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "high",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true, 5*time.Minute),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		}, {
			Namespace: "les",
			Version:   "1.0",
			Service:   NewPrivateLightAPI(&s.lesCommons),
			Public:    false,
		}, {
			Namespace: "vflux",
			Version:   "1.0",
			Service:   s.serverPool.API(),
			Public:    false,
		},
	}...)
}

func (s *LightHighcoin) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightHighcoin) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightHighcoin) TxPool() *light.TxPool              { return s.txPool }
func (s *LightHighcoin) Engine() consensus.Engine           { return s.engine }
func (s *LightHighcoin) LesVersion() int                    { return int(ClientProtocolVersions[0]) }
func (s *LightHighcoin) Downloader() *downloader.Downloader { return s.handler.downloader }
func (s *LightHighcoin) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols returns all the currently configured network protocols to start.
func (s *LightHighcoin) Protocols() []p2p.Protocol {
	return s.makeProtocols(ClientProtocolVersions, s.handler.runPeer, func(id enode.ID) interface{} {
		if p := s.peers.peer(id.String()); p != nil {
			return p.Info()
		}
		return nil
	}, s.dialCandidates)
}

// Start implements node.Lifecycle, starting all internal goroutines needed by the
// light highcoin protocol implementation.
func (s *LightHighcoin) Start() error {
	log.Warn("Light client mode is an experimental feature")

	discovery, err := s.setupDiscovery(s.p2pConfig)
	if err != nil {
		return err
	}
	s.serverPool.AddSource(discovery)
	s.serverPool.Start()
	// Start bloom request workers.
	s.wg.Add(bloomServiceThreads)
	s.startBloomHandlers(params.BloomBitsBlocksClient)
	s.handler.start()

	return nil
}

// Stop implements node.Lifecycle, terminating all internal goroutines used by the
// Highcoin protocol.
func (s *LightHighcoin) Stop() error {
	close(s.closeCh)
	s.serverPool.Stop()
	s.peers.close()
	s.reqDist.close()
	s.odr.Stop()
	s.relay.Stop()
	s.bloomIndexer.Close()
	s.chtIndexer.Close()
	s.blockchain.Stop()
	s.handler.stop()
	s.txPool.Stop()
	s.engine.Close()
	s.pruner.close()
	s.eventMux.Stop()
	rawdb.PopUncleanShutdownMarker(s.chainDb)
	s.chainDb.Close()
	s.lesDb.Close()
	s.wg.Wait()
	log.Info("Light highcoin stopped")
	return nil
}
