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

// Package rift implements the CryptoRift protocol.
package rift

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/cryptorift/riftcore/accounts"
	"github.com/cryptorift/riftcore/common"
	"github.com/cryptorift/riftcore/common/hexutil"
	"github.com/cryptorift/riftcore/consensus"
	"github.com/cryptorift/riftcore/consensus/clique"
	"github.com/cryptorift/riftcore/consensus/rifthash"
	"github.com/cryptorift/riftcore/core"
	"github.com/cryptorift/riftcore/core/types"
	"github.com/cryptorift/riftcore/core/vm"
	"github.com/cryptorift/riftcore/rift/downloader"
	"github.com/cryptorift/riftcore/rift/filters"
	"github.com/cryptorift/riftcore/rift/gasprice"
	"github.com/cryptorift/riftcore/riftdb"
	"github.com/cryptorift/riftcore/event"
	"github.com/cryptorift/riftcore/internal/riftapi"
	"github.com/cryptorift/riftcore/log"
	"github.com/cryptorift/riftcore/miner"
	"github.com/cryptorift/riftcore/node"
	"github.com/cryptorift/riftcore/p2p"
	"github.com/cryptorift/riftcore/params"
	"github.com/cryptorift/riftcore/rlp"
	"github.com/cryptorift/riftcore/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
}

// CryptoRift implements the CryptoRift full node service.
type CryptoRift struct {
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the cryptorift
	stopDbUpgrade func() error // stop chain db sequential key upgrade
	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer
	// DB interfaces
	chainDb riftdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	ApiBackend *RiftApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	riftbase common.Address

	networkId     uint64
	netRPCService *riftapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and riftbase)
}

func (s *CryptoRift) AddLesServer(ls LesServer) {
	s.lesServer = ls
}

// New creates a new CryptoRift object (including the
// initialisation of the common CryptoRift object)
func New(ctx *node.ServiceContext, config *Config) (*CryptoRift, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run rift.CryptoRift in light sync mode, use les.LightCryptorift")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}

	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	rift := &CryptoRift{
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, config, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		riftbase:      config.Riftbase,
	}

	if err := addMipmapBloomBins(chainDb); err != nil {
		return nil, err
	}
	log.Info("Initialising CryptoRift protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run riftcmd upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}

	vmConfig := vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
	rift.blockchain, err = core.NewBlockChain(chainDb, rift.chainConfig, rift.engine, rift.eventMux, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		rift.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	newPool := core.NewTxPool(config.TxPool, rift.chainConfig, rift.EventMux(), rift.blockchain.State, rift.blockchain.GasLimit)
	rift.txPool = newPool

	maxPeers := config.MaxPeers
	if config.LightServ > 0 {
		// if we are running a light server, limit the number of RIFT peers so that we reserve some space for incoming LES connections
		// temporary solution until the new peer connectivity API is finished
		halfPeers := maxPeers / 2
		maxPeers -= config.LightPeers
		if maxPeers < halfPeers {
			maxPeers = halfPeers
		}
	}

	if rift.protocolManager, err = NewProtocolManager(rift.chainConfig, config.SyncMode, config.NetworkId, maxPeers, rift.eventMux, rift.txPool, rift.engine, rift.blockchain, chainDb); err != nil {
		return nil, err
	}

	rift.miner = miner.New(rift, rift.chainConfig, rift.EventMux(), rift.engine)
	rift.miner.SetExtra(makeExtraData(config.ExtraData))

	rift.ApiBackend = &RiftApiBackend{rift, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	rift.ApiBackend.gpo = gasprice.NewOracle(rift.ApiBackend, gpoParams)

	return rift, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"riftcmd",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (riftdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*riftdb.LDBDatabase); ok {
		db.Meter("rift/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an CryptoRift service
func CreateConsensusEngine(ctx *node.ServiceContext, config *Config, chainConfig *params.ChainConfig, db riftdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch {
	case config.PowFake:
		log.Warn("Rifthash used in fake mode")
		return rifthash.NewFaker()
	case config.PowTest:
		log.Warn("Rifthash used in test mode")
		return rifthash.NewTester()
	case config.PowShared:
		log.Warn("Rifthash used in shared mode")
		return rifthash.NewShared()
	default:
		engine := rifthash.New(ctx.ResolvePath(config.RifthashCacheDir), config.RifthashCachesInMem, config.RifthashCachesOnDisk,
			config.RifthashDatasetDir, config.RifthashDatasetsInMem, config.RifthashDatasetsOnDisk)
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the cryptorift package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *CryptoRift) APIs() []rpc.API {
	apis := riftapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "rift",
			Version:   "1.0",
			Service:   NewPublicCryptoriftAPI(s),
			Public:    true,
		}, {
			Namespace: "rift",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "rift",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "rift",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *CryptoRift) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *CryptoRift) Riftbase() (eb common.Address, err error) {
	s.lock.RLock()
	riftbase := s.riftbase
	s.lock.RUnlock()

	if riftbase != (common.Address{}) {
		return riftbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			return accounts[0].Address, nil
		}
	}
	return common.Address{}, fmt.Errorf("riftbase address must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *CryptoRift) SetRiftbase(riftbase common.Address) {
	self.lock.Lock()
	self.riftbase = riftbase
	self.lock.Unlock()

	self.miner.SetRiftbase(riftbase)
}

func (s *CryptoRift) StartMining(local bool) error {
	eb, err := s.Riftbase()
	if err != nil {
		log.Error("Cannot start mining without riftbase", "err", err)
		return fmt.Errorf("riftbase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Riftbase account unavailable locally", "err", err)
			return fmt.Errorf("singer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *CryptoRift) StopMining()         { s.miner.Stop() }
func (s *CryptoRift) IsMining() bool      { return s.miner.Mining() }
func (s *CryptoRift) Miner() *miner.Miner { return s.miner }

func (s *CryptoRift) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *CryptoRift) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *CryptoRift) TxPool() *core.TxPool               { return s.txPool }
func (s *CryptoRift) EventMux() *event.TypeMux           { return s.eventMux }
func (s *CryptoRift) Engine() consensus.Engine           { return s.engine }
func (s *CryptoRift) ChainDb() riftdb.Database            { return s.chainDb }
func (s *CryptoRift) IsListening() bool                  { return true } // Always listening
func (s *CryptoRift) RiftVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *CryptoRift) NetVersion() uint64                 { return s.networkId }
func (s *CryptoRift) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *CryptoRift) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	} else {
		return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
	}
}

// Start implements node.Service, starting all internal goroutines needed by the
// CryptoRift protocol implementation.
func (s *CryptoRift) Start(srvr *p2p.Server) error {
	s.netRPCService = riftapi.NewPublicNetAPI(srvr, s.NetVersion())

	s.protocolManager.Start()
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// CryptoRift protocol.
func (s *CryptoRift) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
