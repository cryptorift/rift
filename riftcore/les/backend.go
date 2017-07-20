// Copyright 2017 The CryptoRift Authors
// This file is part of the riftcore library.
//
// The riftcore library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The riftcore library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the riftcore library. If not, see <http://www.gnu.org/licenses/>.

// Package les implements the Light CryptoRift Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/cryptorift/riftcore/accounts"
	"github.com/cryptorift/riftcore/common"
	"github.com/cryptorift/riftcore/common/hexutil"
	"github.com/cryptorift/riftcore/consensus"
	"github.com/cryptorift/riftcore/core"
	"github.com/cryptorift/riftcore/core/types"
	"github.com/cryptorift/riftcore/rift"
	"github.com/cryptorift/riftcore/rift/downloader"
	"github.com/cryptorift/riftcore/rift/filters"
	"github.com/cryptorift/riftcore/rift/gasprice"
	"github.com/cryptorift/riftcore/riftdb"
	"github.com/cryptorift/riftcore/event"
	"github.com/cryptorift/riftcore/internal/riftapi"
	"github.com/cryptorift/riftcore/light"
	"github.com/cryptorift/riftcore/log"
	"github.com/cryptorift/riftcore/node"
	"github.com/cryptorift/riftcore/p2p"
	"github.com/cryptorift/riftcore/p2p/discv5"
	"github.com/cryptorift/riftcore/params"
	rpc "github.com/cryptorift/riftcore/rpc"
)

type LightCryptorift struct {
	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool
	// Handlers
	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager
	// DB interfaces
	chainDb riftdb.Database // Block chain database

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *riftapi.PublicNetAPI

	quitSync chan struct{}
	wg       sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *rift.Config) (*LightCryptorift, error) {
	chainDb, err := rift.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	rift := &LightCryptorift{
		chainConfig:    chainConfig,
		chainDb:        chainDb,
		eventMux:       ctx.EventMux,
		peers:          peers,
		reqDist:        newRequestDistributor(peers, quitSync),
		accountManager: ctx.AccountManager,
		engine:         rift.CreateConsensusEngine(ctx, config, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		networkId:      config.NetworkId,
	}

	rift.relay = NewLesTxRelay(peers, rift.reqDist)
	rift.serverPool = newServerPool(chainDb, quitSync, &rift.wg)
	rift.retriever = newRetrieveManager(peers, rift.reqDist, rift.serverPool)
	rift.odr = NewLesOdr(chainDb, rift.retriever)
	if rift.blockchain, err = light.NewLightChain(rift.odr, rift.chainConfig, rift.engine, rift.eventMux); err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		rift.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	rift.txPool = light.NewTxPool(rift.chainConfig, rift.eventMux, rift.blockchain, rift.relay)
	if rift.protocolManager, err = NewProtocolManager(rift.chainConfig, true, config.NetworkId, rift.eventMux, rift.engine, rift.peers, rift.blockchain, nil, chainDb, rift.odr, rift.relay, quitSync, &rift.wg); err != nil {
		return nil, err
	}
	rift.ApiBackend = &LesApiBackend{rift, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	rift.ApiBackend.gpo = gasprice.NewOracle(rift.ApiBackend, gpoParams)
	return rift, nil
}

func lesTopic(genesisHash common.Hash) discv5.Topic {
	return discv5.Topic("LES@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// Riftbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Riftbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for Riftbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the cryptorift package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightCryptorift) APIs() []rpc.API {
	return append(riftapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "rift",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "rift",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "rift",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *LightCryptorift) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightCryptorift) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightCryptorift) TxPool() *light.TxPool              { return s.txPool }
func (s *LightCryptorift) Engine() consensus.Engine           { return s.engine }
func (s *LightCryptorift) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *LightCryptorift) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightCryptorift) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *LightCryptorift) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// CryptoRift protocol implementation.
func (s *LightCryptorift) Start(srvr *p2p.Server) error {
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = riftapi.NewPublicNetAPI(srvr, s.networkId)
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash()))
	s.protocolManager.Start()
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// CryptoRift protocol.
func (s *LightCryptorift) Stop() error {
	s.odr.Stop()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
