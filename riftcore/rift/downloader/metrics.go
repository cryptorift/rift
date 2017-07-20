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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/cryptorift/riftcore/metrics"
)

var (
	headerInMeter      = metrics.NewMeter("rift/downloader/headers/in")
	headerReqTimer     = metrics.NewTimer("rift/downloader/headers/req")
	headerDropMeter    = metrics.NewMeter("rift/downloader/headers/drop")
	headerTimeoutMeter = metrics.NewMeter("rift/downloader/headers/timeout")

	bodyInMeter      = metrics.NewMeter("rift/downloader/bodies/in")
	bodyReqTimer     = metrics.NewTimer("rift/downloader/bodies/req")
	bodyDropMeter    = metrics.NewMeter("rift/downloader/bodies/drop")
	bodyTimeoutMeter = metrics.NewMeter("rift/downloader/bodies/timeout")

	receiptInMeter      = metrics.NewMeter("rift/downloader/receipts/in")
	receiptReqTimer     = metrics.NewTimer("rift/downloader/receipts/req")
	receiptDropMeter    = metrics.NewMeter("rift/downloader/receipts/drop")
	receiptTimeoutMeter = metrics.NewMeter("rift/downloader/receipts/timeout")

	stateInMeter   = metrics.NewMeter("rift/downloader/states/in")
	stateDropMeter = metrics.NewMeter("rift/downloader/states/drop")
)
