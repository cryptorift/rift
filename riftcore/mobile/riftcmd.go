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

// Contains all the wrappers from the node package to support client side node
// management on mobile platforms.

package riftcmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/cryptorift/riftcore/core"
	"github.com/cryptorift/riftcore/rift"
	"github.com/cryptorift/riftcore/rift/downloader"
	"github.com/cryptorift/riftcore/riftclient"
	"github.com/cryptorift/riftcore/riftstats"
	"github.com/cryptorift/riftcore/les"
	"github.com/cryptorift/riftcore/node"
	"github.com/cryptorift/riftcore/p2p"
	"github.com/cryptorift/riftcore/p2p/nat"
	"github.com/cryptorift/riftcore/params"
	whisper "github.com/cryptorift/riftcore/whisper/whisperv5"
)

// NodeConfig represents the collection of configuration values to fine tune the Riftcmd
// node embedded into a mobile process. The available values are a subset of the
// entire API provided by riftcore to reduce the maintenance surface and dev
// complexity.
type NodeConfig struct {
	// Bootstrap nodes used to establish connectivity with the rest of the network.
	BootstrapNodes *Enodes

	// MaxPeers is the maximum number of peers that can be connected. If this is
	// set to zero, then only the configured static and trusted peers can connect.
	MaxPeers int

	// CryptoRiftEnabled specifies whether the node should run the CryptoRift protocol.
	CryptoRiftEnabled bool

	// CryptoRiftNetworkID is the network identifier used by the CryptoRift protocol to
	// decide if remote peers should be accepted or not.
	CryptoRiftNetworkID int64 // uint64 in truth, but Java can't handle that...

	// CryptoRiftGenesis is the genesis JSON to use to seed the blockchain with. An
	// empty genesis state is equivalent to using the mainnet's state.
	CryptoRiftGenesis string

	// CryptoRiftDatabaseCache is the system memory in MB to allocate for database caching.
	// A minimum of 16MB is always reserved.
	CryptoRiftDatabaseCache int

	// CryptoRiftNetStats is a netstats connection string to use to report various
	// chain, transaction and node stats to a monitoring server.
	//
	// It has the form "nodename:secret@host:port"
	CryptoRiftNetStats string

	// WhisperEnabled specifies whether the node should run the Whisper protocol.
	WhisperEnabled bool
}

// defaultNodeConfig contains the default node configuration values to use if all
// or some fields are missing from the user's specified list.
var defaultNodeConfig = &NodeConfig{
	BootstrapNodes:        FoundationBootnodes(),
	MaxPeers:              25,
	CryptoRiftEnabled:       true,
	CryptoRiftNetworkID:     1,
	CryptoRiftDatabaseCache: 16,
}

// NewNodeConfig creates a new node option set, initialized to the default values.
func NewNodeConfig() *NodeConfig {
	config := *defaultNodeConfig
	return &config
}

// Node represents a Riftcmd CryptoRift node instance.
type Node struct {
	node *node.Node
}

// NewNode creates and configures a new Riftcmd node.
func NewNode(datadir string, config *NodeConfig) (stack *Node, _ error) {
	// If no or partial configurations were specified, use defaults
	if config == nil {
		config = NewNodeConfig()
	}
	if config.MaxPeers == 0 {
		config.MaxPeers = defaultNodeConfig.MaxPeers
	}
	if config.BootstrapNodes == nil || config.BootstrapNodes.Size() == 0 {
		config.BootstrapNodes = defaultNodeConfig.BootstrapNodes
	}
	// Create the empty networking stack
	nodeConf := &node.Config{
		Name:        clientIdentifier,
		Version:     params.Version,
		DataDir:     datadir,
		KeyStoreDir: filepath.Join(datadir, "keystore"), // Mobile should never use internal keystores!
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
			DiscoveryV5Addr:  ":0",
			BootstrapNodesV5: config.BootstrapNodes.nodes,
			ListenAddr:       ":0",
			NAT:              nat.Any(),
			MaxPeers:         config.MaxPeers,
		},
	}
	rawStack, err := node.New(nodeConf)
	if err != nil {
		return nil, err
	}

	var genesis *core.Genesis
	if config.CryptoRiftGenesis != "" {
		// Parse the user supplied genesis spec if not mainnet
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.CryptoRiftGenesis), genesis); err != nil {
			return nil, fmt.Errorf("invalid genesis spec: %v", err)
		}
		// If we have the testnet, hard code the chain configs too
		if config.CryptoRiftGenesis == TestnetGenesis() {
			genesis.Config = params.TestnetChainConfig
			if config.CryptoRiftNetworkID == 1 {
				config.CryptoRiftNetworkID = 3
			}
		}
	}
	// Register the CryptoRift protocol if requested
	if config.CryptoRiftEnabled {
		riftConf := rift.DefaultConfig
		riftConf.Genesis = genesis
		riftConf.SyncMode = downloader.LightSync
		riftConf.NetworkId = uint64(config.CryptoRiftNetworkID)
		riftConf.DatabaseCache = config.CryptoRiftDatabaseCache
		if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, &riftConf)
		}); err != nil {
			return nil, fmt.Errorf("cryptorift init: %v", err)
		}
		// If netstats reporting is requested, do it
		if config.CryptoRiftNetStats != "" {
			if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
				var lesServ *les.LightCryptorift
				ctx.Service(&lesServ)

				return riftstats.New(config.CryptoRiftNetStats, nil, lesServ)
			}); err != nil {
				return nil, fmt.Errorf("netstats init: %v", err)
			}
		}
	}
	// Register the Whisper protocol if requested
	if config.WhisperEnabled {
		if err := rawStack.Register(func(*node.ServiceContext) (node.Service, error) {
			return whisper.New(&whisper.DefaultConfig), nil
		}); err != nil {
			return nil, fmt.Errorf("whisper init: %v", err)
		}
	}
	return &Node{rawStack}, nil
}

// Start creates a live P2P node and starts running it.
func (n *Node) Start() error {
	return n.node.Start()
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	return n.node.Stop()
}

// GetCryptoriftClient retrieves a client to access the CryptoRift subsystem.
func (n *Node) GetCryptoriftClient() (client *CryptoRiftClient, _ error) {
	rpc, err := n.node.Attach()
	if err != nil {
		return nil, err
	}
	return &CryptoRiftClient{riftclient.NewClient(rpc)}, nil
}

// GetNodeInfo gathers and returns a collection of metadata known about the host.
func (n *Node) GetNodeInfo() *NodeInfo {
	return &NodeInfo{n.node.Server().NodeInfo()}
}

// GetPeersInfo returns an array of metadata objects describing connected peers.
func (n *Node) GetPeersInfo() *PeerInfos {
	return &PeerInfos{n.node.Server().PeersInfo()}
}
