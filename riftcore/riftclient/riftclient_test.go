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

package riftclient

import "github.com/cryptorift/riftcore"

// Verify that Client implements the cryptorift interfaces.
var (
	_ = cryptorift.ChainReader(&Client{})
	_ = cryptorift.TransactionReader(&Client{})
	_ = cryptorift.ChainStateReader(&Client{})
	_ = cryptorift.ChainSyncReader(&Client{})
	_ = cryptorift.ContractCaller(&Client{})
	_ = cryptorift.GasEstimator(&Client{})
	_ = cryptorift.GasPricer(&Client{})
	_ = cryptorift.LogFilterer(&Client{})
	_ = cryptorift.PendingStateReader(&Client{})
	// _ = cryptorift.PendingStateEventer(&Client{})
	_ = cryptorift.PendingContractCaller(&Client{})
)
