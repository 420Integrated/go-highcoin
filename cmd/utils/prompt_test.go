// Copyright 2020 The go-highcoin Authors
// This file is part of go-highcoin.
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

// Package utils contains internal helper functions for go-highcoin commands.
package utils

import (
	"testing"
)

func TestGetPassPhraseWithList(t *testing.T) {
	type args struct {
		text         string
		confirmation bool
		index        int
		passwords    []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"test1",
			args{
				"text1",
				false,
				0,
				[]string{"zero", "one", "two"},
			},
			"zero",
		},
		{
			"test2",
			args{
				"text2",
				false,
				5,
				[]string{"zero", "one", "two"},
			},
			"two",
		},
		{
			"test3",
			args{
				"text3",
				true,
				1,
				[]string{"zero", "one", "two"},
			},
			"one",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPassPhraseWithList(tt.args.text, tt.args.confirmation, tt.args.index, tt.args.passwords); got != tt.want {
				t.Errorf("GetPassPhraseWithList() = %v, want %v", got, tt.want)
			}
		})
	}
}
