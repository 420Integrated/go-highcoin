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

package core

import (
	"fmt"

	"github.com/420integrated/go-highcoin/consensus"
	"github.com/420integrated/go-highcoin/core/state"
	"github.com/420integrated/go-highcoin/core/types"
	"github.com/420integrated/go-highcoin/params"
	"github.com/420integrated/go-highcoin/trie"
)

// BlockValidator is responsible for validating block headers, uncles and
// processed state.
//
// BlockValidator implements Validator.
type BlockValidator struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for validating
}

// NewBlockValidator returns a new block validator which is safe for re-use
func NewBlockValidator(config *params.ChainConfig, blockchain *BlockChain, engine consensus.Engine) *BlockValidator {
	validator := &BlockValidator{
		config: config,
		engine: engine,
		bc:     blockchain,
	}
	return validator
}

// ValidateBody validates the given block's uncles and verifies the block
// header's transaction and uncle roots. The headers are assumed to be already
// validated at this point.
func (v *BlockValidator) ValidateBody(block *types.Block) error {
	// Check if the block's known, and if not, that it's linkable
	if v.bc.HasBlockAndState(block.Hash(), block.NumberU64()) {
		return ErrKnownBlock
	}
	// Header validity is known at this point, check the uncles and transactions
	header := block.Header()
	if err := v.engine.VerifyUncles(v.bc, block); err != nil {
		return err
	}
	if hash := types.CalcUncleHash(block.Uncles()); hash != header.UncleHash {
		return fmt.Errorf("uncle root hash mismatch: have %x, want %x", hash, header.UncleHash)
	}
	if hash := types.DeriveSha(block.Transactions(), trie.NewStackTrie(nil)); hash != header.TxHash {
		return fmt.Errorf("transaction root hash mismatch: have %x, want %x", hash, header.TxHash)
	}
	if !v.bc.HasBlockAndState(block.ParentHash(), block.NumberU64()-1) {
		if !v.bc.HasBlock(block.ParentHash(), block.NumberU64()-1) {
			return consensus.ErrUnknownAncestor
		}
		return consensus.ErrPrunedAncestor
	}
	return nil
}

// ValidateState validates the various changes that happen after a state
// transition, such as amount of used smoke, the receipt roots and the state root
// itself. ValidateState returns a database batch if the validation was a success
// otherwise nil and an error is returned.
func (v *BlockValidator) ValidateState(block *types.Block, statedb *state.StateDB, receipts types.Receipts, usedSmoke uint64) error {
	header := block.Header()
	if block.SmokeUsed() != usedSmoke {
		return fmt.Errorf("invalid smoke used (remote: %d local: %d)", block.SmokeUsed(), usedSmoke)
	}
	// Validate the received block's bloom with the one derived from the generated receipts.
	// For valid blocks this should always validate to true.
	rbloom := types.CreateBloom(receipts)
	if rbloom != header.Bloom {
		return fmt.Errorf("invalid bloom (remote: %x  local: %x)", header.Bloom, rbloom)
	}
	// Tre receipt Trie's root (R = (Tr [[H1, R1], ... [Hn, Rn]]))
	receiptSha := types.DeriveSha(receipts, trie.NewStackTrie(nil))
	if receiptSha != header.ReceiptHash {
		return fmt.Errorf("invalid receipt root hash (remote: %x local: %x)", header.ReceiptHash, receiptSha)
	}
	// Validate the state root against the received state root and throw
	// an error if they don't match.
	if root := statedb.IntermediateRoot(v.config.IsEIP158(header.Number)); header.Root != root {
		return fmt.Errorf("invalid merkle root (remote: %x local: %x)", header.Root, root)
	}
	return nil
}

// CalcSmokeLimit computes the smoke limit of the next block after parent. It aims
// to keep the baseline smoke above the provided floor, and increase it towards the
// ceil if the blocks are full. If the ceil is exceeded, it will always decrease
// the smoke allowance.
func CalcSmokeLimit(parent *types.Block, smokeFloor, smokeCeil uint64) uint64 {
	// contrib = (parentSmokeUsed * 3 / 2) / 1024
	contrib := (parent.SmokeUsed() + parent.SmokeUsed()/2) / params.SmokeLimitBoundDivisor

	// decay = parentSmokeLimit / 1024 -1
	decay := parent.SmokeLimit()/params.SmokeLimitBoundDivisor - 1

	/*
		strategy: smokeLimit of block-to-mine is set based on parent's
		smokeUsed value.  if parentSmokeUsed > parentSmokeLimit * (2/3) then we
		increase it, otherwise lower it (or leave it unchanged if it's right
		at that usage) the amount increased/decreased depends on how far away
		from parentSmokeLimit * (2/3) parentSmokeUsed is.
	*/
	limit := parent.SmokeLimit() - decay + contrib
	if limit < params.MinSmokeLimit {
		limit = params.MinSmokeLimit
	}
	// If we're outside our allowed smoke range, we try to hone towards them
	if limit < smokeFloor {
		limit = parent.SmokeLimit() + decay
		if limit > smokeFloor {
			limit = smokeFloor
		}
	} else if limit > smokeCeil {
		limit = parent.SmokeLimit() - decay
		if limit < smokeCeil {
			limit = smokeCeil
		}
	}
	return limit
}
