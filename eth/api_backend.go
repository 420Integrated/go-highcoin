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

package eth

import (
	"context"
	"errors"
	"math/big"

	"github.com/420integrated/go-highcoin/accounts"
	"github.com/420integrated/go-highcoin/common"
	"github.com/420integrated/go-highcoin/consensus"
	"github.com/420integrated/go-highcoin/core"
	"github.com/420integrated/go-highcoin/core/bloombits"
	"github.com/420integrated/go-highcoin/core/rawdb"
	"github.com/420integrated/go-highcoin/core/state"
	"github.com/420integrated/go-highcoin/core/types"
	"github.com/420integrated/go-highcoin/core/vm"
	"github.com/420integrated/go-highcoin/eth/downloader"
	"github.com/420integrated/go-highcoin/eth/gasprice"
	"github.com/420integrated/go-highcoin/ethdb"
	"github.com/420integrated/go-highcoin/event"
	"github.com/420integrated/go-highcoin/miner"
	"github.com/420integrated/go-highcoin/params"
	"github.com/420integrated/go-highcoin/rpc"
)

// HighAPIBackend implements ethapi.Backend for full nodes
type HighAPIBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	eth                 *Highcoin
	gpo                 *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *HighAPIBackend) ChainConfig() *params.ChainConfig {
	return b.high.blockchain.Config()
}

func (b *HighAPIBackend) CurrentBlock() *types.Block {
	return b.high.blockchain.CurrentBlock()
}

func (b *HighAPIBackend) SetHead(number uint64) {
	b.high.handler.downloader.Cancel()
	b.high.blockchain.SetHead(number)
}

func (b *HighAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.high.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.high.blockchain.CurrentBlock().Header(), nil
	}
	return b.high.blockchain.GetHeaderByNumber(uint64(number)), nil
}

func (b *HighAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.high.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.high.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *HighAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.high.blockchain.GetHeaderByHash(hash), nil
}

func (b *HighAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.high.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.high.blockchain.CurrentBlock(), nil
	}
	return b.high.blockchain.GetBlockByNumber(uint64(number)), nil
}

func (b *HighAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.high.blockchain.GetBlockByHash(hash), nil
}

func (b *HighAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.high.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.high.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.high.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *HighAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, state := b.high.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.high.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *HighAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.high.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, err := b.high.BlockChain().StateAt(header.Root)
		return stateDb, header, err
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *HighAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.high.blockchain.GetReceiptsByHash(hash), nil
}

func (b *HighAPIBackend) GetLogs(ctx context.Context, hash common.Hash) ([][]*types.Log, error) {
	receipts := b.high.blockchain.GetReceiptsByHash(hash)
	if receipts == nil {
		return nil, nil
	}
	logs := make([][]*types.Log, len(receipts))
	for i, receipt := range receipts {
		logs[i] = receipt.Logs
	}
	return logs, nil
}

func (b *HighAPIBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	return b.high.blockchain.GetTdByHash(hash)
}

func (b *HighAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header) (*vm.EVM, func() error, error) {
	vmError := func() error { return nil }

	txContext := core.NewEVMTxContext(msg)
	context := core.NewEVMBlockContext(header, b.high.BlockChain(), nil)
	return vm.NewEVM(context, txContext, state, b.high.blockchain.Config(), *b.high.blockchain.GetVMConfig()), vmError, nil
}

func (b *HighAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.high.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *HighAPIBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.high.miner.SubscribePendingLogs(ch)
}

func (b *HighAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.high.BlockChain().SubscribeChainEvent(ch)
}

func (b *HighAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.high.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *HighAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.high.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *HighAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.high.BlockChain().SubscribeLogsEvent(ch)
}

func (b *HighAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.high.txPool.AddLocal(signedTx)
}

func (b *HighAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.high.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *HighAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.high.txPool.Get(hash)
}

func (b *HighAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.high.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *HighAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.high.txPool.Nonce(addr), nil
}

func (b *HighAPIBackend) Stats() (pending int, queued int) {
	return b.high.txPool.Stats()
}

func (b *HighAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.high.TxPool().Content()
}

func (b *HighAPIBackend) TxPool() *core.TxPool {
	return b.high.TxPool()
}

func (b *HighAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.high.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *HighAPIBackend) Downloader() *downloader.Downloader {
	return b.high.Downloader()
}

func (b *HighAPIBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *HighAPIBackend) ChainDb() ethdb.Database {
	return b.high.ChainDb()
}

func (b *HighAPIBackend) EventMux() *event.TypeMux {
	return b.high.EventMux()
}

func (b *HighAPIBackend) AccountManager() *accounts.Manager {
	return b.high.AccountManager()
}

func (b *HighAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *HighAPIBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *HighAPIBackend) RPCGasCap() uint64 {
	return b.high.config.RPCGasCap
}

func (b *HighAPIBackend) RPCTxFeeCap() float64 {
	return b.high.config.RPCTxFeeCap
}

func (b *HighAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.high.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *HighAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.high.bloomRequests)
	}
}

func (b *HighAPIBackend) Engine() consensus.Engine {
	return b.high.engine
}

func (b *HighAPIBackend) CurrentHeader() *types.Header {
	return b.high.blockchain.CurrentHeader()
}

func (b *HighAPIBackend) Miner() *miner.Miner {
	return b.high.Miner()
}

func (b *HighAPIBackend) StartMining(threads int) error {
	return b.high.StartMining(threads)
}

func (b *HighAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64) (*state.StateDB, func(), error) {
	return b.high.stateAtBlock(block, reexec)
}

func (b *HighAPIBackend) StatesInRange(ctx context.Context, fromBlock *types.Block, toBlock *types.Block, reexec uint64) ([]*state.StateDB, func(), error) {
	return b.high.statesInRange(fromBlock, toBlock, reexec)
}

func (b *HighAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, func(), error) {
	return b.high.stateAtTransaction(block, txIndex, reexec)
}
