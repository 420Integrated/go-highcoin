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

package high

import (
	"github.com/420integrated/go-highcoin/core"
	"github.com/420integrated/go-highcoin/core/forkid"
	"github.com/420integrated/go-highcoin/p2p/dnsdisc"
	"github.com/420integrated/go-highcoin/p2p/enode"
	"github.com/420integrated/go-highcoin/rlp"
)

// highEntry is the "high" ENR entry which advertises high protocol
// on the discovery network.
type highEntry struct {
	ForkID forkid.ID // Fork identifier per EIP-2124

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

// ENRKey implements enr.Entry.
func (e highEntry) ENRKey() string {
	return "high"
}

// startHighEntryUpdate starts the ENR updater loop.
func (high *Highcoin) startHighEntryUpdate(ln *enode.LocalNode) {
	var newHead = make(chan core.ChainHeadEvent, 10)
	sub := high.blockchain.SubscribeChainHeadEvent(newHead)

	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case <-newHead:
				ln.Set(high.currentHighEntry())
			case <-sub.Err():
				// Would be nice to sync with high.Stop, but there is no
				// good way to do that.
				return
			}
		}
	}()
}

func (high *Highcoin) currentHighEntry() *highEntry {
	return &highEntry{ForkID: forkid.NewID(high.blockchain.Config(), high.blockchain.Genesis().Hash(),
		high.blockchain.CurrentHeader().Number.Uint64())}
}

// setupDiscovery creates the node discovery source for the `high` and `snap`
// protocols.
func setupDiscovery(urls []string) (enode.Iterator, error) {
	if len(urls) == 0 {
		return nil, nil
	}
	client := dnsdisc.NewClient(dnsdisc.Config{})
	return client.NewIterator(urls...)
}
