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
	"math/big"
	"testing"

	"github.com/cryptorift/riftcore/common"
	"github.com/cryptorift/riftcore/core"
	"github.com/cryptorift/riftcore/core/types"
	"github.com/cryptorift/riftcore/riftdb"
	"github.com/cryptorift/riftcore/params"
)

func TestMipmapUpgrade(t *testing.T) {
	db, _ := riftdb.NewMemDatabase()
	addr := common.BytesToAddress([]byte("jeff"))
	genesis := new(core.Genesis).MustCommit(db)

	chain, receipts := core.GenerateChain(params.TestChainConfig, genesis, db, 10, func(i int, gen *core.BlockGen) {
		switch i {
		case 1:
			receipt := types.NewReceipt(nil, new(big.Int))
			receipt.Logs = []*types.Log{{Address: addr}}
			gen.AddUncheckedReceipt(receipt)
		case 2:
			receipt := types.NewReceipt(nil, new(big.Int))
			receipt.Logs = []*types.Log{{Address: addr}}
			gen.AddUncheckedReceipt(receipt)
		}
	})
	for i, block := range chain {
		core.WriteBlock(db, block)
		if err := core.WriteCanonicalHash(db, block.Hash(), block.NumberU64()); err != nil {
			t.Fatalf("failed to insert block number: %v", err)
		}
		if err := core.WriteHeadBlockHash(db, block.Hash()); err != nil {
			t.Fatalf("failed to insert block number: %v", err)
		}
		if err := core.WriteBlockReceipts(db, block.Hash(), block.NumberU64(), receipts[i]); err != nil {
			t.Fatal("error writing block receipts:", err)
		}
	}

	err := addMipmapBloomBins(db)
	if err != nil {
		t.Fatal(err)
	}

	bloom := core.GetMipmapBloom(db, 1, core.MIPMapLevels[0])
	if (bloom == types.Bloom{}) {
		t.Error("got empty bloom filter")
	}

	data, _ := db.Get([]byte("setting-mipmap-version"))
	if len(data) == 0 {
		t.Error("setting-mipmap-version not written to database")
	}
}
