// Copyright 2017 The go-highcoin Authors
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

package params

// These are the multipliers for highcoin denominations.
// Example: To get the marleys value of an amount in 'gmarleys', use
//
//    new(big.Int).Mul(value, big.NewInt(params.GMarleys))
//
const (
	Marleys   = 1
	GMarleys  = 1e9
	Highcoin = 1e18
)
