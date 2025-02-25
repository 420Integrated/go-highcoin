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
	"crypto/rand"
	"math/big"

	"github.com/420integrated/go-highcoin/common"
	"github.com/420integrated/go-highcoin/common/hexutil"
	"github.com/420integrated/go-highcoin/core/types"
)

// largeNumber returns a very large big.Int.
func largeNumber(megabytes int) *big.Int {
	buf := make([]byte, megabytes*1024*1024)
	rand.Read(buf)
	bigint := new(big.Int)
	bigint.SetBytes(buf)
	return bigint
}

// largeBuffer returns a very large buffer.
func largeBuffer(megabytes int) []byte {
	buf := make([]byte, megabytes*1024*1024)
	rand.Read(buf)
	return buf
}

// largeString returns a very large string.
func largeString(megabytes int) string {
	buf := make([]byte, megabytes*1024*1024)
	rand.Read(buf)
	return hexutil.Encode(buf)
}

func largeBlock() *types.Block {
	return types.NewBlockWithHeader(largeHeader())
}

// Returns a random hash
func randHash() common.Hash {
	var h common.Hash
	rand.Read(h[:])
	return h
}

func largeHeader() *types.Header {
	return &types.Header{
		MixDigest:   randHash(),
		ReceiptHash: randHash(),
		TxHash:      randHash(),
		Nonce:       types.BlockNonce{},
		Extra:       []byte{},
		Bloom:       types.Bloom{},
		SmokeUsed:     0,
		Coinbase:    common.Address{},
		SmokeLimit:    0,
		UncleHash:   randHash(),
		Time:        1337,
		ParentHash:  randHash(),
		Root:        randHash(),
		Number:      largeNumber(2),
		Difficulty:  largeNumber(2),
	}
}
