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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/420integrated/go-highcoin/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("high/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("high/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("high/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("high/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("high/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("high/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("high/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("high/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("high/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("high/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("high/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("high/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("high/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("high/downloader/states/drop", nil)

	throttleCounter = metrics.NewRegisteredCounter("high/downloader/throttle", nil)
)
