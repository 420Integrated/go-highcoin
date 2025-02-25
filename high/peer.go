// Copyright 2015 The go-highcoin Authors
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
	"math/big"
	"sync"
	"time"

	"github.com/420integrated/go-highcoin/high/protocols/high"
	"github.com/420integrated/go-highcoin/high/protocols/snap"
)

// highPeerInfo represents a short summary of the `high` sub-protocol metadata known
// about a connected peer.
type highPeerInfo struct {
	Version    uint     `json:"version"`    // Highcoin protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // Hex hash of the peer's best owned block
}

// highPeer is a wrapper around high.Peer to maintain a few extra metadata.
type highPeer struct {
	*high.Peer
	snapExt *snapPeer // Satellite `snap` connection

	syncDrop *time.Timer   // Connection dropper if `high` sync progress isn't validated in time
	snapWait chan struct{} // Notification channel for snap connections
	lock     sync.RWMutex  // Mutex protecting the internal fields
}

// info gathers and returns some `high` protocol metadata known about a peer.
func (p *highPeer) info() *highPeerInfo {
	hash, td := p.Head()

	return &highPeerInfo{
		Version:    p.Version(),
		Difficulty: td,
		Head:       hash.Hex(),
	}
}

// snapPeerInfo represents a short summary of the `snap` sub-protocol metadata known
// about a connected peer.
type snapPeerInfo struct {
	Version uint `json:"version"` // Snapshot protocol version negotiated
}

// snapPeer is a wrapper around snap.Peer to maintain a few extra metadata.
type snapPeer struct {
	*snap.Peer
}

// info gathers and returns some `snap` protocol metadata known about a peer.
func (p *snapPeer) info() *snapPeerInfo {
	return &snapPeerInfo{
		Version: p.Version(),
	}
}
