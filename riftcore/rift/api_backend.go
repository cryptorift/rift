// Copyright 2017 The CryptoRift Authors
// This file is part of the riftcore library.
//
// The riftcore library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The riftcore library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the riftcore library. If not, see <http://www.gnu.org/licenses/>.

package rift

import (
	"context"
	"math/big"

	"github.com/cryptorift/riftcore/accounts"
	"github.com/cryptorift/riftcore/common"
	"github.com/cryptorift/riftcore/common/math"
	"github.com/cryptorift/riftcore/core"
	"github.com/cryptorift/riftcore/core/state"
	"github.com/cryptorift/riftcore/core/types"
	"github.com/cryptorift/riftcore/core/vm"
	"github.com/cryptorift/riftcore/rift/downloader"
	"github.com/cryptorift/riftcore/rift/gasprice"
	"github.com/cryptorift/riftcore/riftdb"
	"github.com/cryptorift/riftcore/event"
	"github.com/cryptorift/riftcore/params"
	"github.com/cryptorift/riftcore/rpc"
)

// RiftApiBackend implements riftapi.Backend for full nodes
type RiftApiBackend struct {
	rift *CryptoRift
	gpo *gasprice.Oracle
}

func (b *RiftApiBackend) ChainConfig() *params.ChainConfig {
	return b.rift.chainConfig
}

func (b *RiftApiBackend) CurrentBlock() *types.Block {
	return b.rift.blockchain.CurrentBlock()
}

func (b *RiftApiBackend) SetHead(number uint64) {
	b.rift.protocolManager.downloader.Cancel()
	b.rift.blockchain.SetHead(number)
}

func (b *RiftApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.rift.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.rift.blockchain.CurrentBlock().Header(), nil
	}
	return b.rift.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *RiftApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.rift.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.rift.blockchain.CurrentBlock(), nil
	}
	return b.rift.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *RiftApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.rift.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.rift.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *RiftApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.rift.blockchain.GetBlockByHash(blockHash), nil
}

func (b *RiftApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.rift.chainDb, blockHash, core.GetBlockNumber(b.rift.chainDb, blockHash)), nil
}

func (b *RiftApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.rift.blockchain.GetTdByHash(blockHash)
}

func (b *RiftApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.rift.BlockChain(), nil)
	return vm.NewEVM(context, state, b.rift.chainConfig, vmCfg), vmError, nil
}

func (b *RiftApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.rift.txPool.AddLocal(signedTx)
}

func (b *RiftApiBackend) RemoveTx(txHash common.Hash) {
	b.rift.txPool.Remove(txHash)
}

func (b *RiftApiBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.rift.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *RiftApiBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.rift.txPool.Get(hash)
}

func (b *RiftApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.rift.txPool.State().GetNonce(addr), nil
}

func (b *RiftApiBackend) Stats() (pending int, queued int) {
	return b.rift.txPool.Stats()
}

func (b *RiftApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.rift.TxPool().Content()
}

func (b *RiftApiBackend) Downloader() *downloader.Downloader {
	return b.rift.Downloader()
}

func (b *RiftApiBackend) ProtocolVersion() int {
	return b.rift.RiftVersion()
}

func (b *RiftApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *RiftApiBackend) ChainDb() riftdb.Database {
	return b.rift.ChainDb()
}

func (b *RiftApiBackend) EventMux() *event.TypeMux {
	return b.rift.EventMux()
}

func (b *RiftApiBackend) AccountManager() *accounts.Manager {
	return b.rift.AccountManager()
}
