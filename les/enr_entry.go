// Copyright 2019 The go-highcoin Authors
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

package les

import (
	"github.com/420integrated/go-highcoin/core/forkid"
	"github.com/420integrated/go-highcoin/p2p"
	"github.com/420integrated/go-highcoin/p2p/dnsdisc"
	"github.com/420integrated/go-highcoin/p2p/enode"
	"github.com/420integrated/go-highcoin/rlp"
)

// lesEntry is the "les" ENR entry. This is set for LES servers only.
type lesEntry struct {
	// Ignore additional fields (for forward compatibility).
	_ []rlp.RawValue `rlp:"tail"`
}

func (lesEntry) ENRKey() string { return "les" }

// highEntry is the "high" ENR entry. This is redeclared here to avoid depending on package high.
type highEntry struct {
	ForkID forkid.ID
	_      []rlp.RawValue `rlp:"tail"`
}

func (highEntry) ENRKey() string { return "high" }

// setupDiscovery creates the node discovery source for the high protocol.
func (high *LightHighcoin) setupDiscovery(cfg *p2p.Config) (enode.Iterator, error) {
	it := enode.NewFairMix(0)

	// Enable DNS discovery.
	if len(high.config.HighDiscoveryURLs) != 0 {
		client := dnsdisc.NewClient(dnsdisc.Config{})
		dns, err := client.NewIterator(high.config.HighDiscoveryURLs...)
		if err != nil {
			return nil, err
		}
		it.AddSource(dns)
	}

	// Enable DHT.
	if cfg.DiscoveryV5 && high.p2pServer.DiscV5 != nil {
		it.AddSource(high.p2pServer.DiscV5.RandomNodes())
	}

	forkFilter := forkid.NewFilter(high.blockchain)
	iterator := enode.Filter(it, func(n *enode.Node) bool { return nodeIsServer(forkFilter, n) })
	return iterator, nil
}

// nodeIsServer checks if n is an LES server node.
func nodeIsServer(forkFilter forkid.Filter, n *enode.Node) bool {
	var les lesEntry
	var high highEntry
	return n.Load(&les) == nil && n.Load(&high) == nil && forkFilter(high.ForkID) == nil
}
