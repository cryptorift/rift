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

// Contains the metrics collected by the fetcher.

package fetcher

import (
	"github.com/cryptorift/riftcore/metrics"
)

var (
	propAnnounceInMeter   = metrics.NewMeter("rift/fetcher/prop/announces/in")
	propAnnounceOutTimer  = metrics.NewTimer("rift/fetcher/prop/announces/out")
	propAnnounceDropMeter = metrics.NewMeter("rift/fetcher/prop/announces/drop")
	propAnnounceDOSMeter  = metrics.NewMeter("rift/fetcher/prop/announces/dos")

	propBroadcastInMeter   = metrics.NewMeter("rift/fetcher/prop/broadcasts/in")
	propBroadcastOutTimer  = metrics.NewTimer("rift/fetcher/prop/broadcasts/out")
	propBroadcastDropMeter = metrics.NewMeter("rift/fetcher/prop/broadcasts/drop")
	propBroadcastDOSMeter  = metrics.NewMeter("rift/fetcher/prop/broadcasts/dos")

	headerFetchMeter = metrics.NewMeter("rift/fetcher/fetch/headers")
	bodyFetchMeter   = metrics.NewMeter("rift/fetcher/fetch/bodies")

	headerFilterInMeter  = metrics.NewMeter("rift/fetcher/filter/headers/in")
	headerFilterOutMeter = metrics.NewMeter("rift/fetcher/filter/headers/out")
	bodyFilterInMeter    = metrics.NewMeter("rift/fetcher/filter/bodies/in")
	bodyFilterOutMeter   = metrics.NewMeter("rift/fetcher/filter/bodies/out")
)
