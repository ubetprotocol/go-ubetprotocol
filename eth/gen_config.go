// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package eth

import (
	"math/big"
	"time"

	"github.com/ubetprotocol/go-ubetprotocol/common"
	"github.com/ubetprotocol/go-ubetprotocol/consensus/ethash"
	"github.com/ubetprotocol/go-ubetprotocol/core"
	"github.com/ubetprotocol/go-ubetprotocol/eth/downloader"
	"github.com/ubetprotocol/go-ubetprotocol/eth/gasprice"
	"github.com/ubetprotocol/go-ubetprotocol/miner"
	"github.com/ubetprotocol/go-ubetprotocol/params"
)

// MarshalTOML marshals as TOML.
func (c Config) MarshalTOML() (interface{}, error) {
	type Config struct {
		Genesis                 *core.Genesis `toml:",omitempty"`
		NetworkId               uint64
		SyncMode                downloader.SyncMode
		NoPruning               bool
		NoPrefetch              bool
		Whitelist               map[uint64]common.Hash `toml:"-"`
		LightServ               int                    `toml:",omitempty"`
		LightBandwidthIn        int                    `toml:",omitempty"`
		LightBandwidthOut       int                    `toml:",omitempty"`
		LightPeers              int                    `toml:",omitempty"`
		OnlyAnnounce            bool
		ULC                     *ULCConfig `toml:",omitempty"`
		SkipBcVersionCheck      bool       `toml:"-"`
		DatabaseHandles         int        `toml:"-"`
		DatabaseCache           int
		DatabaseFreezer         string
		TrieCleanCache          int
		TrieDirtyCache          int
		TrieTimeout             time.Duration
		Miner                   miner.Config
		Ethash                  ethash.Config
		TxPool                  core.TxPoolConfig
		GPO                     gasprice.Config
		EnablePreimageRecording bool
		DocRoot                 string `toml:"-"`
		EWASMInterpreter        string
		EVMInterpreter          string
		ConstantinopleOverride  *big.Int
		RPCGasCap               *big.Int `toml:",omitempty"`
		Checkpoint              *params.TrustedCheckpoint
		CheckpointOracle        *params.CheckpointOracleConfig
	}
	var enc Config
	enc.Genesis = c.Genesis
	enc.NetworkId = c.NetworkId
	enc.SyncMode = c.SyncMode
	enc.NoPruning = c.NoPruning
	enc.NoPrefetch = c.NoPrefetch
	enc.Whitelist = c.Whitelist
	enc.LightServ = c.LightServ
	enc.LightBandwidthIn = c.LightBandwidthIn
	enc.LightBandwidthOut = c.LightBandwidthOut
	enc.LightPeers = c.LightPeers
	enc.OnlyAnnounce = c.OnlyAnnounce
	enc.ULC = c.ULC
	enc.SkipBcVersionCheck = c.SkipBcVersionCheck
	enc.DatabaseHandles = c.DatabaseHandles
	enc.DatabaseCache = c.DatabaseCache
	enc.DatabaseFreezer = c.DatabaseFreezer
	enc.TrieCleanCache = c.TrieCleanCache
	enc.TrieDirtyCache = c.TrieDirtyCache
	enc.TrieTimeout = c.TrieTimeout
	enc.Miner = c.Miner
	enc.Ethash = c.Ethash
	enc.TxPool = c.TxPool
	enc.GPO = c.GPO
	enc.EnablePreimageRecording = c.EnablePreimageRecording
	enc.DocRoot = c.DocRoot
	enc.EWASMInterpreter = c.EWASMInterpreter
	enc.EVMInterpreter = c.EVMInterpreter
	enc.ConstantinopleOverride = c.ConstantinopleOverride
	enc.RPCGasCap = c.RPCGasCap
	enc.Checkpoint = c.Checkpoint
	enc.CheckpointOracle = c.CheckpointOracle
	return &enc, nil
}

// UnmarshalTOML unmarshals from TOML.
func (c *Config) UnmarshalTOML(unmarshal func(interface{}) error) error {
	type Config struct {
		Genesis                 *core.Genesis `toml:",omitempty"`
		NetworkId               *uint64
		SyncMode                *downloader.SyncMode
		NoPruning               *bool
		NoPrefetch              *bool
		Whitelist               map[uint64]common.Hash `toml:"-"`
		LightServ               *int                   `toml:",omitempty"`
		LightBandwidthIn        *int                   `toml:",omitempty"`
		LightBandwidthOut       *int                   `toml:",omitempty"`
		LightPeers              *int                   `toml:",omitempty"`
		OnlyAnnounce            *bool
		ULC                     *ULCConfig `toml:",omitempty"`
		SkipBcVersionCheck      *bool      `toml:"-"`
		DatabaseHandles         *int       `toml:"-"`
		DatabaseCache           *int
		DatabaseFreezer         *string
		TrieCleanCache          *int
		TrieDirtyCache          *int
		TrieTimeout             *time.Duration
		Miner                   *miner.Config
		Ethash                  *ethash.Config
		TxPool                  *core.TxPoolConfig
		GPO                     *gasprice.Config
		EnablePreimageRecording *bool
		DocRoot                 *string `toml:"-"`
		EWASMInterpreter        *string
		EVMInterpreter          *string
		ConstantinopleOverride  *big.Int
		RPCGasCap               *big.Int `toml:",omitempty"`
		Checkpoint              *params.TrustedCheckpoint
		CheckpointOracle        *params.CheckpointOracleConfig
	}
	var dec Config
	if err := unmarshal(&dec); err != nil {
		return err
	}
	if dec.Genesis != nil {
		c.Genesis = dec.Genesis
	}
	if dec.NetworkId != nil {
		c.NetworkId = *dec.NetworkId
	}
	if dec.SyncMode != nil {
		c.SyncMode = *dec.SyncMode
	}
	if dec.NoPruning != nil {
		c.NoPruning = *dec.NoPruning
	}
	if dec.NoPrefetch != nil {
		c.NoPrefetch = *dec.NoPrefetch
	}
	if dec.Whitelist != nil {
		c.Whitelist = dec.Whitelist
	}
	if dec.LightServ != nil {
		c.LightServ = *dec.LightServ
	}
	if dec.LightBandwidthIn != nil {
		c.LightBandwidthIn = *dec.LightBandwidthIn
	}
	if dec.LightBandwidthOut != nil {
		c.LightBandwidthOut = *dec.LightBandwidthOut
	}
	if dec.LightPeers != nil {
		c.LightPeers = *dec.LightPeers
	}
	if dec.OnlyAnnounce != nil {
		c.OnlyAnnounce = *dec.OnlyAnnounce
	}
	if dec.ULC != nil {
		c.ULC = dec.ULC
	}
	if dec.SkipBcVersionCheck != nil {
		c.SkipBcVersionCheck = *dec.SkipBcVersionCheck
	}
	if dec.DatabaseHandles != nil {
		c.DatabaseHandles = *dec.DatabaseHandles
	}
	if dec.DatabaseCache != nil {
		c.DatabaseCache = *dec.DatabaseCache
	}
	if dec.DatabaseFreezer != nil {
		c.DatabaseFreezer = *dec.DatabaseFreezer
	}
	if dec.TrieCleanCache != nil {
		c.TrieCleanCache = *dec.TrieCleanCache
	}
	if dec.TrieDirtyCache != nil {
		c.TrieDirtyCache = *dec.TrieDirtyCache
	}
	if dec.TrieTimeout != nil {
		c.TrieTimeout = *dec.TrieTimeout
	}
	if dec.Miner != nil {
		c.Miner = *dec.Miner
	}
	if dec.Ethash != nil {
		c.Ethash = *dec.Ethash
	}
	if dec.TxPool != nil {
		c.TxPool = *dec.TxPool
	}
	if dec.GPO != nil {
		c.GPO = *dec.GPO
	}
	if dec.EnablePreimageRecording != nil {
		c.EnablePreimageRecording = *dec.EnablePreimageRecording
	}
	if dec.DocRoot != nil {
		c.DocRoot = *dec.DocRoot
	}
	if dec.EWASMInterpreter != nil {
		c.EWASMInterpreter = *dec.EWASMInterpreter
	}
	if dec.EVMInterpreter != nil {
		c.EVMInterpreter = *dec.EVMInterpreter
	}
	if dec.ConstantinopleOverride != nil {
		c.ConstantinopleOverride = dec.ConstantinopleOverride
	}
	if dec.RPCGasCap != nil {
		c.RPCGasCap = dec.RPCGasCap
	}
	if dec.Checkpoint != nil {
		c.Checkpoint = dec.Checkpoint
	}
	if dec.CheckpointOracle != nil {
		c.CheckpointOracle = dec.CheckpointOracle
	}
	return nil
}
