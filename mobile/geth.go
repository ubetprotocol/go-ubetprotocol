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

// Contains all the wrappers from the node package to support client side node
// management on mobile platforms.

package gubet

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/ubetprotocol/go-ubetprotocol/core"
	"github.com/ubetprotocol/go-ubetprotocol/eth"
	"github.com/ubetprotocol/go-ubetprotocol/eth/downloader"
	"github.com/ubetprotocol/go-ubetprotocol/ethclient"
	"github.com/ubetprotocol/go-ubetprotocol/ethstats"
	"github.com/ubetprotocol/go-ubetprotocol/internal/debug"
	"github.com/ubetprotocol/go-ubetprotocol/les"
	"github.com/ubetprotocol/go-ubetprotocol/node"
	"github.com/ubetprotocol/go-ubetprotocol/p2p"
	"github.com/ubetprotocol/go-ubetprotocol/p2p/nat"
	"github.com/ubetprotocol/go-ubetprotocol/params"
	whisper "github.com/ubetprotocol/go-ubetprotocol/whisper/whisperv6"
)

// NodeConfig represents the collection of configuration values to fine tune the Gubet
// node embedded into a mobile process. The available values are a subset of the
// entire API provided by go-ubetprotocol to reduce the maintenance surface and dev
// complexity.
type NodeConfig struct {
	// Bootstrap nodes used to establish connectivity with the rest of the network.
	BootstrapNodes *Enodes

	// MaxPeers is the maximum number of peers that can be connected. If this is
	// set to zero, then only the configured static and trusted peers can connect.
	MaxPeers int

	// UbetprotocolEnabled specifies whether the node should run the Ubetprotocol protocol.
	UbetprotocolEnabled bool

	// UbetprotocolNetworkID is the network identifier used by the Ubetprotocol protocol to
	// decide if remote peers should be accepted or not.
	UbetprotocolNetworkID int64 // uint64 in truth, but Java can't handle that...

	// UbetprotocolGenesis is the genesis JSON to use to seed the blockchain with. An
	// empty genesis state is equivalent to using the mainnet's state.
	UbetprotocolGenesis string

	// UbetprotocolDatabaseCache is the system memory in MB to allocate for database caching.
	// A minimum of 16MB is always reserved.
	UbetprotocolDatabaseCache int

	// UbetprotocolNetStats is a netstats connection string to use to report various
	// chain, transaction and node stats to a monitoring server.
	//
	// It has the form "nodename:secret@host:port"
	UbetprotocolNetStats string

	// WhisperEnabled specifies whether the node should run the Whisper protocol.
	WhisperEnabled bool

	// Listening address of pprof server.
	PprofAddress string

	// Ultra Light client options
	ULC *eth.ULCConfig
}

// defaultNodeConfig contains the default node configuration values to use if all
// or some fields are missing from the user's specified list.
var defaultNodeConfig = &NodeConfig{
	BootstrapNodes:        FoundationBootnodes(),
	MaxPeers:              25,
	UbetprotocolEnabled:       true,
	UbetprotocolNetworkID:     1,
	UbetprotocolDatabaseCache: 16,
}

// NewNodeConfig creates a new node option set, initialized to the default values.
func NewNodeConfig() *NodeConfig {
	config := *defaultNodeConfig
	return &config
}

// Node represents a Gubet Ubetprotocol node instance.
type Node struct {
	node *node.Node
}

// NewNode creates and configures a new Gubet node.
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

	if config.PprofAddress != "" {
		debug.StartPProf(config.PprofAddress)
	}

	// Create the empty networking stack
	nodeConf := &node.Config{
		Name:        clientIdentifier,
		Version:     params.VersionWithMeta,
		DataDir:     datadir,
		KeyStoreDir: filepath.Join(datadir, "keystore"), // Mobile should never use internal keystores!
		P2P: p2p.Config{
			NoDiscovery:      true,
			DiscoveryV5:      true,
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

	debug.Memsize.Add("node", rawStack)

	var genesis *core.Genesis
	if config.UbetprotocolGenesis != "" {
		// Parse the user supplied genesis spec if not mainnet
		genesis = new(core.Genesis)
		if err := json.Unmarshal([]byte(config.UbetprotocolGenesis), genesis); err != nil {
			return nil, fmt.Errorf("invalid genesis spec: %v", err)
		}
		// If we have the testnet, hard code the chain configs too
		if config.UbetprotocolGenesis == TestnetGenesis() {
			genesis.Config = params.TestnetChainConfig
			if config.UbetprotocolNetworkID == 1 {
				config.UbetprotocolNetworkID = 3
			}
		}
	}
	// Register the Ubetprotocol protocol if requested
	if config.UbetprotocolEnabled {
		ethConf := eth.DefaultConfig
		ethConf.Genesis = genesis
		ethConf.SyncMode = downloader.LightSync
		ethConf.NetworkId = uint64(config.UbetprotocolNetworkID)
		ethConf.DatabaseCache = config.UbetprotocolDatabaseCache
		if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, &ethConf)
		}); err != nil {
			return nil, fmt.Errorf("ubetprotocol init: %v", err)
		}
		// If netstats reporting is requested, do it
		if config.UbetprotocolNetStats != "" {
			if err := rawStack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
				var lesServ *les.LightUbetprotocol
				ctx.Service(&lesServ)

				return ethstats.New(config.UbetprotocolNetStats, nil, lesServ)
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

// Close terminates a running node along with all it's services, tearing internal
// state doen too. It's not possible to restart a closed node.
func (n *Node) Close() error {
	return n.node.Close()
}

// Start creates a live P2P node and starts running it.
func (n *Node) Start() error {
	return n.node.Start()
}

// Stop terminates a running node along with all it's services. If the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	return n.node.Stop()
}

// GetUbetprotocolClient retrieves a client to access the Ubetprotocol subsystem.
func (n *Node) GetUbetprotocolClient() (client *UbetprotocolClient, _ error) {
	rpc, err := n.node.Attach()
	if err != nil {
		return nil, err
	}
	return &UbetprotocolClient{ethclient.NewClient(rpc)}, nil
}

// GetNodeInfo gathers and returns a collection of metadata known about the host.
func (n *Node) GetNodeInfo() *NodeInfo {
	return &NodeInfo{n.node.Server().NodeInfo()}
}

// GetPeersInfo returns an array of metadata objects describing connected peers.
func (n *Node) GetPeersInfo() *PeerInfos {
	return &PeerInfos{n.node.Server().PeersInfo()}
}
