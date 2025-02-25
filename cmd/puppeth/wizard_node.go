// Copyright 2017 The go-highcoin Authors
// This file is part of go-highcoin.
//
// go-highcoin is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-highcoin is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-highcoin. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/420integrated/go-highcoin/accounts/keystore"
	"github.com/420integrated/go-highcoin/common"
	"github.com/420integrated/go-highcoin/log"
)

// deployNode creates a new node configuration based on some user input.
func (w *wizard) deployNode(boot bool) {
	// Do some sanity check before the user wastes time on input
	if w.conf.Genesis == nil {
		log.Error("No genesis block configured")
		return
	}
	if w.conf.highstats == "" {
		log.Error("No highstats server configured")
		return
	}
	// Select the server to interact with
	server := w.selectServer()
	if server == "" {
		return
	}
	client := w.servers[server]

	// Retrieve any active node configurations from the server
	infos, err := checkNode(client, w.network, boot)
	if err != nil {
		if boot {
			infos = &nodeInfos{port: 30303, peersTotal: 512, peersLight: 256}
		} else {
			infos = &nodeInfos{port: 30303, peersTotal: 50, peersLight: 0, smokeTarget: 7.5, smokeLimit: 10, smokePrice: 1}
		}
	}
	existed := err == nil

	infos.genesis, _ = json.MarshalIndent(w.conf.Genesis, "", "  ")
	infos.network = w.conf.Genesis.Config.ChainID.Int64()

	// Figure out where the user wants to store the persistent data
	fmt.Println()
	if infos.datadir == "" {
		fmt.Printf("Where should data be stored on the remote machine?\n")
		infos.datadir = w.readString()
	} else {
		fmt.Printf("Where should data be stored on the remote machine? (default = %s)\n", infos.datadir)
		infos.datadir = w.readDefaultString(infos.datadir)
	}
	if w.conf.Genesis.Config.Ethash != nil && !boot {
		fmt.Println()
		if infos.ethashdir == "" {
			fmt.Printf("Where should the ethash mining DAGs be stored on the remote machine?\n")
			infos.ethashdir = w.readString()
		} else {
			fmt.Printf("Where should the ethash mining DAGs be stored on the remote machine? (default = %s)\n", infos.ethashdir)
			infos.ethashdir = w.readDefaultString(infos.ethashdir)
		}
	}
	// Figure out which port to listen on
	fmt.Println()
	fmt.Printf("Which TCP/UDP port to listen on? (default = %d)\n", infos.port)
	infos.port = w.readDefaultInt(infos.port)

	// Figure out how many peers to allow (different based on node type)
	fmt.Println()
	fmt.Printf("How many peers to allow connecting? (default = %d)\n", infos.peersTotal)
	infos.peersTotal = w.readDefaultInt(infos.peersTotal)

	// Figure out how many light peers to allow (different based on node type)
	fmt.Println()
	fmt.Printf("How many light peers to allow connecting? (default = %d)\n", infos.peersLight)
	infos.peersLight = w.readDefaultInt(infos.peersLight)

	// Set a proper name to report on the stats page
	fmt.Println()
	if infos.highstats == "" {
		fmt.Printf("What should the node be called on the stats page?\n")
		infos.highstats = w.readString() + ":" + w.conf.highstats
	} else {
		fmt.Printf("What should the node be called on the stats page? (default = %s)\n", infos.highstats)
		infos.highstats = w.readDefaultString(infos.highstats) + ":" + w.conf.highstats
	}
	// If the node is a miner/signer, load up needed credentials
	if !boot {
		if w.conf.Genesis.Config.Ethash != nil {
			// Ethash based miners only need an highcoinbase to mine against
			fmt.Println()
			if infos.highcoinbase == "" {
				fmt.Printf("What address should the miner use?\n")
				for {
					if address := w.readAddress(); address != nil {
						infos.highcoinbase = address.Hex()
						break
					}
				}
			} else {
				fmt.Printf("What address should the miner use? (default = %s)\n", infos.highcoinbase)
				infos.highcoinbase = w.readDefaultAddress(common.HexToAddress(infos.highcoinbase)).Hex()
			}
		} else if w.conf.Genesis.Config.Clique != nil {
			// If a previous signer was already set, offer to reuse it
			if infos.keyJSON != "" {
				if key, err := keystore.DecryptKey([]byte(infos.keyJSON), infos.keyPass); err != nil {
					infos.keyJSON, infos.keyPass = "", ""
				} else {
					fmt.Println()
					fmt.Printf("Reuse previous (%s) signing account (y/n)? (default = yes)\n", key.Address.Hex())
					if !w.readDefaultYesNo(true) {
						infos.keyJSON, infos.keyPass = "", ""
					}
				}
			}
			// Clique based signers need a keyfile and unlock password, ask if unavailable
			if infos.keyJSON == "" {
				fmt.Println()
				fmt.Println("Please paste the signer's key JSON:")
				infos.keyJSON = w.readJSON()

				fmt.Println()
				fmt.Println("What's the unlock password for the account? (won't be echoed)")
				infos.keyPass = w.readPassword()

				if _, err := keystore.DecryptKey([]byte(infos.keyJSON), infos.keyPass); err != nil {
					log.Error("Failed to decrypt key with given password")
					return
				}
			}
		}
		// Establish the smoke dynamics to be enforced by the signer
		fmt.Println()
		fmt.Printf("What smoke limit should empty blocks target (MSmoke)? (default = %0.3f)\n", infos.smokeTarget)
		infos.smokeTarget = w.readDefaultFloat(infos.smokeTarget)

		fmt.Println()
		fmt.Printf("What smoke limit should full blocks target (MSmoke)? (default = %0.3f)\n", infos.smokeLimit)
		infos.smokeLimit = w.readDefaultFloat(infos.smokeLimit)

		fmt.Println()
		fmt.Printf("What smoke price should the signer require (GMarleys)? (default = %0.3f)\n", infos.smokePrice)
		infos.smokePrice = w.readDefaultFloat(infos.smokePrice)
	}
	// Try to deploy the full node on the host
	nocache := false
	if existed {
		fmt.Println()
		fmt.Printf("Should the node be built from scratch (y/n)? (default = no)\n")
		nocache = w.readDefaultYesNo(false)
	}
	if out, err := deployNode(client, w.network, w.conf.bootnodes, infos, nocache); err != nil {
		log.Error("Failed to deploy Highcoin node container", "err", err)
		if len(out) > 0 {
			fmt.Printf("%s\n", out)
		}
		return
	}
	// All ok, run a network scan to pick any changes up
	log.Info("Waiting for node to finish booting")
	time.Sleep(3 * time.Second)

	w.networkStats()
}
