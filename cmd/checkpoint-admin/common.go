// Copyright 2018 The go-ubetprotocol Authors
// This file is part of go-ubetprotocol.
//
// go-ubetprotocol is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ubetprotocol is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ubetprotocol. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/ubetprotocol/go-ubetprotocol/accounts/keystore"
	"github.com/ubetprotocol/go-ubetprotocol/cmd/utils"
	"github.com/ubetprotocol/go-ubetprotocol/common"
	"github.com/ubetprotocol/go-ubetprotocol/console"
	"github.com/ubetprotocol/go-ubetprotocol/contracts/checkpointoracle"
	"github.com/ubetprotocol/go-ubetprotocol/ethclient"
	"github.com/ubetprotocol/go-ubetprotocol/params"
	"github.com/ubetprotocol/go-ubetprotocol/rpc"
	"gopkg.in/urfave/cli.v1"
)

// newClient creates a client with specified remote URL.
func newClient(ctx *cli.Context) *ethclient.Client {
	client, err := ethclient.Dial(ctx.GlobalString(nodeURLFlag.Name))
	if err != nil {
		utils.Fatalf("Failed to connect to Ethereum node: %v", err)
	}
	return client
}

// newRPCClient creates a rpc client with specified node URL.
func newRPCClient(url string) *rpc.Client {
	client, err := rpc.Dial(url)
	if err != nil {
		utils.Fatalf("Failed to connect to Ethereum node: %v", err)
	}
	return client
}

// getContractAddr retrieves the register contract address through
// rpc request.
func getContractAddr(client *rpc.Client) common.Address {
	var addr string
	if err := client.Call(&addr, "les_getCheckpointContractAddress"); err != nil {
		utils.Fatalf("Failed to fetch checkpoint oracle address: %v", err)
	}
	return common.HexToAddress(addr)
}

// getCheckpoint retrieves the specified checkpoint or the latest one
// through rpc request.
func getCheckpoint(ctx *cli.Context, client *rpc.Client) *params.TrustedCheckpoint {
	var checkpoint *params.TrustedCheckpoint

	if ctx.GlobalIsSet(indexFlag.Name) {
		var result [3]string
		index := uint64(ctx.GlobalInt64(indexFlag.Name))
		if err := client.Call(&result, "les_getCheckpoint", index); err != nil {
			utils.Fatalf("Failed to get local checkpoint %v, please ensure the les API is exposed", err)
		}
		checkpoint = &params.TrustedCheckpoint{
			SectionIndex: index,
			SectionHead:  common.HexToHash(result[0]),
			CHTRoot:      common.HexToHash(result[1]),
			BloomRoot:    common.HexToHash(result[2]),
		}
	} else {
		var result [4]string
		err := client.Call(&result, "les_latestCheckpoint")
		if err != nil {
			utils.Fatalf("Failed to get local checkpoint %v, please ensure the les API is exposed", err)
		}
		index, err := strconv.ParseUint(result[0], 0, 64)
		if err != nil {
			utils.Fatalf("Failed to parse checkpoint index %v", err)
		}
		checkpoint = &params.TrustedCheckpoint{
			SectionIndex: index,
			SectionHead:  common.HexToHash(result[1]),
			CHTRoot:      common.HexToHash(result[2]),
			BloomRoot:    common.HexToHash(result[3]),
		}
	}
	return checkpoint
}

// newContract creates a registrar contract instance with specified
// contract address or the default contracts for mainnet or testnet.
func newContract(client *rpc.Client) (common.Address, *checkpointoracle.CheckpointOracle) {
	addr := getContractAddr(client)
	if addr == (common.Address{}) {
		utils.Fatalf("No specified registrar contract address")
	}
	contract, err := checkpointoracle.NewCheckpointOracle(addr, ethclient.NewClient(client))
	if err != nil {
		utils.Fatalf("Failed to setup registrar contract %s: %v", addr, err)
	}
	return addr, contract
}

// promptPassphrase prompts the user for a passphrase.
// Set confirmation to true to require the user to confirm the passphrase.
func promptPassphrase(confirmation bool) string {
	passphrase, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}

	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			utils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if passphrase != confirm {
			utils.Fatalf("Passphrases do not match")
		}
	}
	return passphrase
}

// getPassphrase obtains a passphrase given by the user. It first checks the
// --password command line flag and ultimately prompts the user for a
// passphrase.
func getPassphrase(ctx *cli.Context) string {
	passphraseFile := ctx.String(utils.PasswordFileFlag.Name)
	if passphraseFile != "" {
		content, err := ioutil.ReadFile(passphraseFile)
		if err != nil {
			utils.Fatalf("Failed to read passphrase file '%s': %v",
				passphraseFile, err)
		}
		return strings.TrimRight(string(content), "\r\n")
	}
	// Otherwise prompt the user for the passphrase.
	return promptPassphrase(false)
}

// getKey retrieves the user key through specified key file.
func getKey(ctx *cli.Context) *keystore.Key {
	// Read key from file.
	keyFile := ctx.GlobalString(keyFileFlag.Name)
	keyJson, err := ioutil.ReadFile(keyFile)
	if err != nil {
		utils.Fatalf("Failed to read the keyfile at '%s': %v", keyFile, err)
	}
	// Decrypt key with passphrase.
	passphrase := getPassphrase(ctx)
	key, err := keystore.DecryptKey(keyJson, passphrase)
	if err != nil {
		utils.Fatalf("Failed to decrypt user key '%s': %v", keyFile, err)
	}
	return key
}
