// Copyright 2020 The go-highcoin Authors
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

package hightest

import (
	"fmt"
	"net"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/420integrated/go-highcoin/core/types"
	"github.com/420integrated/go-highcoin/crypto"
	"github.com/420integrated/go-highcoin/high/protocols/high"
	"github.com/420integrated/go-highcoin/internal/utesting"
	"github.com/420integrated/go-highcoin/p2p"
	"github.com/420integrated/go-highcoin/p2p/enode"
	"github.com/420integrated/go-highcoin/p2p/rlpx"
	"github.com/stretchr/testify/assert"
)

var pretty = spew.ConfigState{
	Indent:                  "  ",
	DisableCapacities:       true,
	DisablePointerAddresses: true,
	SortKeys:                true,
}

var timeout = 20 * time.Second

// Suite represents a structure used to test the high
// protocol of a node(s).
type Suite struct {
	Dest *enode.Node

	chain     *Chain
	fullChain *Chain
}

// NewSuite creates and returns a new high-test suite that can
// be used to test the given node against the given blockchain
// data.
func NewSuite(dest *enode.Node, chainfile string, genesisfile string) (*Suite, error) {
	chain, err := loadChain(chainfile, genesisfile)
	if err != nil {
		return nil, err
	}
	return &Suite{
		Dest:      dest,
		chain:     chain.Shorten(1000),
		fullChain: chain,
	}, nil
}

func (s *Suite) HighTests() []utesting.Test {
	return []utesting.Test{
		// status
		{Name: "Status", Fn: s.TestStatus},
		{Name: "Status_66", Fn: s.TestStatus_66},
		// get block headers
		{Name: "GetBlockHeaders", Fn: s.TestGetBlockHeaders},
		{Name: "GetBlockHeaders_66", Fn: s.TestGetBlockHeaders_66},
		{Name: "TestSimultaneousRequests_66", Fn: s.TestSimultaneousRequests_66},
		{Name: "TestSameRequestID_66", Fn: s.TestSameRequestID_66},
		{Name: "TestZeroRequestID_66", Fn: s.TestZeroRequestID_66},
		// get block bodies
		{Name: "GetBlockBodies", Fn: s.TestGetBlockBodies},
		{Name: "GetBlockBodies_66", Fn: s.TestGetBlockBodies_66},
		// broadcast
		{Name: "Broadcast", Fn: s.TestBroadcast},
		{Name: "Broadcast_66", Fn: s.TestBroadcast_66},
		{Name: "TestLargeAnnounce", Fn: s.TestLargeAnnounce},
		{Name: "TestLargeAnnounce_66", Fn: s.TestLargeAnnounce_66},
		// malicious handshakes + status
		{Name: "TestMaliciousHandshake", Fn: s.TestMaliciousHandshake},
		{Name: "TestMaliciousStatus", Fn: s.TestMaliciousStatus},
		{Name: "TestMaliciousHandshake_66", Fn: s.TestMaliciousHandshake_66},
		{Name: "TestMaliciousStatus_66", Fn: s.TestMaliciousStatus},
		// test transactions
		{Name: "TestTransactions", Fn: s.TestTransaction},
		{Name: "TestTransactions_66", Fn: s.TestTransaction_66},
		{Name: "TestMaliciousTransactions", Fn: s.TestMaliciousTx},
		{Name: "TestMaliciousTransactions_66", Fn: s.TestMaliciousTx_66},
	}
}

// TestStatus attempts to connect to the given node and exchange
// a status message with it, and then check to make sure
// the chain head is correct.
func (s *Suite) TestStatus(t *utesting.T) {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}
	// get protoHandshake
	conn.handshake(t)
	// get status
	switch msg := conn.statusExchange(t, s.chain, nil).(type) {
	case *Status:
		t.Logf("got status message: %s", pretty.Sdump(msg))
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
}

// TestMaliciousStatus sends a status package with a large total difficulty.
func (s *Suite) TestMaliciousStatus(t *utesting.T) {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}
	// get protoHandshake
	conn.handshake(t)
	status := &Status{
		ProtocolVersion: uint32(conn.highProtocolVersion),
		NetworkID:       s.chain.chainConfig.ChainID.Uint64(),
		TD:              largeNumber(2),
		Head:            s.chain.blocks[s.chain.Len()-1].Hash(),
		Genesis:         s.chain.blocks[0].Hash(),
		ForkID:          s.chain.ForkID(),
	}
	// get status
	switch msg := conn.statusExchange(t, s.chain, status).(type) {
	case *Status:
		t.Logf("%+v\n", msg)
	default:
		t.Fatalf("expected status, got: %#v ", msg)
	}
	// wait for disconnect
	switch msg := conn.ReadAndServe(s.chain, timeout).(type) {
	case *Disconnect:
	case *Error:
		return
	default:
		t.Fatalf("expected disconnect, got: %s", pretty.Sdump(msg))
	}
}

// TestGetBlockHeaders tests if the given node can respond to
// a `GetBlockHeaders` request and that the response is accurate.
func (s *Suite) TestGetBlockHeaders(t *utesting.T) {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}

	conn.handshake(t)
	conn.statusExchange(t, s.chain, nil)

	// get block headers
	req := &GetBlockHeaders{
		Origin: high.HashOrNumber{
			Hash: s.chain.blocks[1].Hash(),
		},
		Amount:  2,
		Skip:    1,
		Reverse: false,
	}

	if err := conn.Write(req); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}

	switch msg := conn.ReadAndServe(s.chain, timeout).(type) {
	case *BlockHeaders:
		headers := *msg
		for _, header := range headers {
			num := header.Number.Uint64()
			t.Logf("received header (%d): %s", num, pretty.Sdump(header.Hash()))
			assert.Equal(t, s.chain.blocks[int(num)].Header(), header)
		}
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
}

// TestGetBlockBodies tests if the given node can respond to
// a `GetBlockBodies` request and that the response is accurate.
func (s *Suite) TestGetBlockBodies(t *utesting.T) {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}

	conn.handshake(t)
	conn.statusExchange(t, s.chain, nil)
	// create block bodies request
	req := &GetBlockBodies{
		s.chain.blocks[54].Hash(),
		s.chain.blocks[75].Hash(),
	}
	if err := conn.Write(req); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}

	switch msg := conn.ReadAndServe(s.chain, timeout).(type) {
	case *BlockBodies:
		t.Logf("received %d block bodies", len(*msg))
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
}

// TestBroadcast tests if a block announcement is correctly
// propagated to the given node's peer(s).
func (s *Suite) TestBroadcast(t *utesting.T) {
	sendConn, receiveConn := s.setupConnection(t), s.setupConnection(t)
	nextBlock := len(s.chain.blocks)
	blockAnnouncement := &NewBlock{
		Block: s.fullChain.blocks[nextBlock],
		TD:    s.fullChain.TD(nextBlock + 1),
	}
	s.testAnnounce(t, sendConn, receiveConn, blockAnnouncement)
	// update test suite chain
	s.chain.blocks = append(s.chain.blocks, s.fullChain.blocks[nextBlock])
	// wait for client to update its chain
	if err := receiveConn.waitForBlock(s.chain.Head()); err != nil {
		t.Fatal(err)
	}
}

// TestMaliciousHandshake tries to send malicious data during the handshake.
func (s *Suite) TestMaliciousHandshake(t *utesting.T) {
	conn, err := s.dial()
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}
	// write hello to client
	pub0 := crypto.FromECDSAPub(&conn.ourKey.PublicKey)[1:]
	handshakes := []*Hello{
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: largeString(2), Version: 64},
			},
			ID: pub0,
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: "high", Version: 64},
				{Name: "high", Version: 65},
			},
			ID: append(pub0, byte(0)),
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: "high", Version: 64},
				{Name: "high", Version: 65},
			},
			ID: append(pub0, pub0...),
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: "high", Version: 64},
				{Name: "high", Version: 65},
			},
			ID: largeBuffer(2),
		},
		{
			Version: 5,
			Caps: []p2p.Cap{
				{Name: largeString(2), Version: 64},
			},
			ID: largeBuffer(2),
		},
	}
	for i, handshake := range handshakes {
		t.Logf("Testing malicious handshake %v\n", i)
		// Init the handshake
		if err := conn.Write(handshake); err != nil {
			t.Fatalf("could not write to connection: %v", err)
		}
		// check that the peer disconnected
		timeout := 20 * time.Second
		// Discard one hello
		for i := 0; i < 2; i++ {
			switch msg := conn.ReadAndServe(s.chain, timeout).(type) {
			case *Disconnect:
			case *Error:
			case *Hello:
				// Hello's are send concurrently, so ignore them
				continue
			default:
				t.Fatalf("unexpected: %s", pretty.Sdump(msg))
			}
		}
		// Dial for the next round
		conn, err = s.dial()
		if err != nil {
			t.Fatalf("could not dial: %v", err)
		}
	}
}

// TestLargeAnnounce tests the announcement mechanism with a large block.
func (s *Suite) TestLargeAnnounce(t *utesting.T) {
	nextBlock := len(s.chain.blocks)
	blocks := []*NewBlock{
		{
			Block: largeBlock(),
			TD:    s.fullChain.TD(nextBlock + 1),
		},
		{
			Block: s.fullChain.blocks[nextBlock],
			TD:    largeNumber(2),
		},
		{
			Block: largeBlock(),
			TD:    largeNumber(2),
		},
		{
			Block: s.fullChain.blocks[nextBlock],
			TD:    s.fullChain.TD(nextBlock + 1),
		},
	}

	for i, blockAnnouncement := range blocks[0:3] {
		t.Logf("Testing malicious announcement: %v\n", i)
		sendConn := s.setupConnection(t)
		if err := sendConn.Write(blockAnnouncement); err != nil {
			t.Fatalf("could not write to connection: %v", err)
		}
		// Invalid announcement, check that peer disconnected
		switch msg := sendConn.ReadAndServe(s.chain, timeout).(type) {
		case *Disconnect:
		case *Error:
			break
		default:
			t.Fatalf("unexpected: %s wanted disconnect", pretty.Sdump(msg))
		}
	}
	// Test the last block as a valid block
	sendConn := s.setupConnection(t)
	receiveConn := s.setupConnection(t)
	s.testAnnounce(t, sendConn, receiveConn, blocks[3])
	// update test suite chain
	s.chain.blocks = append(s.chain.blocks, s.fullChain.blocks[nextBlock])
	// wait for client to update its chain
	if err := receiveConn.waitForBlock(s.fullChain.blocks[nextBlock]); err != nil {
		t.Fatal(err)
	}
}

func (s *Suite) testAnnounce(t *utesting.T, sendConn, receiveConn *Conn, blockAnnouncement *NewBlock) {
	// Announce the block.
	if err := sendConn.Write(blockAnnouncement); err != nil {
		t.Fatalf("could not write to connection: %v", err)
	}
	s.waitAnnounce(t, receiveConn, blockAnnouncement)
}

func (s *Suite) waitAnnounce(t *utesting.T, conn *Conn, blockAnnouncement *NewBlock) {
	timeout := 20 * time.Second
	switch msg := conn.ReadAndServe(s.chain, timeout).(type) {
	case *NewBlock:
		t.Logf("received NewBlock message: %s", pretty.Sdump(msg.Block))
		assert.Equal(t,
			blockAnnouncement.Block.Header(), msg.Block.Header(),
			"wrong block header in announcement",
		)
		assert.Equal(t,
			blockAnnouncement.TD, msg.TD,
			"wrong TD in announcement",
		)
	case *NewBlockHashes:
		message := *msg
		t.Logf("received NewBlockHashes message: %s", pretty.Sdump(message))
		assert.Equal(t, blockAnnouncement.Block.Hash(), message[0].Hash,
			"wrong block hash in announcement",
		)
	default:
		t.Fatalf("unexpected: %s", pretty.Sdump(msg))
	}
}

func (s *Suite) setupConnection(t *utesting.T) *Conn {
	// create conn
	sendConn, err := s.dial()
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}
	sendConn.handshake(t)
	sendConn.statusExchange(t, s.chain, nil)
	return sendConn
}

// dial attempts to dial the given node and perform a handshake,
// returning the created Conn if successful.
func (s *Suite) dial() (*Conn, error) {
	var conn Conn
	// dial
	fd, err := net.Dial("tcp", fmt.Sprintf("%v:%d", s.Dest.IP(), s.Dest.TCP()))
	if err != nil {
		return nil, err
	}
	conn.Conn = rlpx.NewConn(fd, s.Dest.Pubkey())
	// do encHandshake
	conn.ourKey, _ = crypto.GenerateKey()
	_, err = conn.Handshake(conn.ourKey)
	if err != nil {
		return nil, err
	}
	// set default p2p capabilities
	conn.caps = []p2p.Cap{
		{Name: "high", Version: 64},
		{Name: "high", Version: 65},
	}
	return &conn, nil
}

func (s *Suite) TestTransaction(t *utesting.T) {
	tests := []*types.Transaction{
		getNextTxFromChain(t, s),
		unknownTx(t, s),
	}
	for i, tx := range tests {
		t.Logf("Testing tx propagation: %v\n", i)
		sendSuccessfulTx(t, s, tx)
	}
}

func (s *Suite) TestMaliciousTx(t *utesting.T) {
	tests := []*types.Transaction{
		getOldTxFromChain(t, s),
		invalidNonceTx(t, s),
		hugeAmount(t, s),
		hugeSmokePrice(t, s),
		hugeData(t, s),
	}
	for i, tx := range tests {
		t.Logf("Testing malicious tx propagation: %v\n", i)
		sendFailingTx(t, s, tx)
	}
}
