// Copyright (c) 2017-2021 420Integrated Devlopment Team
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

//+build windows

package netutil

import (
	"net"
	"os"
	"syscall"
)

const _WSAEMSGSIZE = syscall.Errno(10040)

// isPacketTooBig reports if err indicates that a UDP packet didn't
// fit the receive buffer. On Windows, WSARecvFrom returns
// code WSAEMSGSIZE and no data if this happens.
func isPacketTooBig(err error) bool {
	if opErr, ok := err.(*net.OpError); ok {
		if scErr, ok := opErr.Err.(*os.SyscallError); ok {
			return scErr.Err == _WSAEMSGSIZE
		}
		return opErr.Err == _WSAEMSGSIZE
	}
	return false
}
