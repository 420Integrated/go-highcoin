// Copyright 2019 The go-highcoin Authors
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

// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package rpc

/*
#include <sys/un.h>

int max_socket_path_size() {
struct sockaddr_un s;
return sizeof(s.sun_path);
}
*/
import "C"

var (
	max_path_size = C.max_socket_path_size()
)
