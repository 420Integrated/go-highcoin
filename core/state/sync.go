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

package state

import (
	"bytes"

	"github.com/420integrated/go-highcoin/common"
	"github.com/420integrated/go-highcoin/highdb"
	"github.com/420integrated/go-highcoin/rlp"
	"github.com/420integrated/go-highcoin/trie"
)

// NewStateSync create a new state trie download scheduler.
func NewStateSync(root common.Hash, database highdb.KeyValueReader, bloom *trie.SyncBloom) *trie.Sync {
	var syncer *trie.Sync
	callback := func(path []byte, leaf []byte, parent common.Hash) error {
		var obj Account
		if err := rlp.Decode(bytes.NewReader(leaf), &obj); err != nil {
			return err
		}
		syncer.AddSubTrie(obj.Root, path, parent, nil)
		syncer.AddCodeEntry(common.BytesToHash(obj.CodeHash), path, parent)
		return nil
	}
	syncer = trie.NewSync(root, database, callback, bloom)
	return syncer
}
