// Copyright 2018 The go-highcoin Authors
// This file is part of go-highcoin.
//
// go-highcoin is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-highcoin is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-highcoin. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/420integrated/go-highcoin/core"
)

// Tests the go-highcoin to Alhigh chainspec conversion for the Stureby testnet.
func TestAlhighSturebyConverter(t *testing.T) {
	blob, err := ioutil.ReadFile("testdata/stureby_highcoin.json")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}
	var genesis core.Genesis
	if err := json.Unmarshal(blob, &genesis); err != nil {
		t.Fatalf("failed parsing genesis: %v", err)
	}
	spec, err := newAlhighGenesisSpec("stureby", &genesis)
	if err != nil {
		t.Fatalf("failed creating chainspec: %v", err)
	}

	expBlob, err := ioutil.ReadFile("testdata/stureby_alhigh.json")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}
	expspec := &alhighGenesisSpec{}
	if err := json.Unmarshal(expBlob, expspec); err != nil {
		t.Fatalf("failed parsing genesis: %v", err)
	}
	if !reflect.DeepEqual(expspec, spec) {
		t.Errorf("chainspec mismatch")
		c := spew.ConfigState{
			DisablePointerAddresses: true,
			SortKeys:                true,
		}
		exp := strings.Split(c.Sdump(expspec), "\n")
		got := strings.Split(c.Sdump(spec), "\n")
		for i := 0; i < len(exp) && i < len(got); i++ {
			if exp[i] != got[i] {
				t.Logf("got: %v\nexp: %v\n", exp[i], got[i])
			}
		}
	}
}

// Tests the go-highcoin to Parity chainspec conversion for the Stureby testnet.
func TestParitySturebyConverter(t *testing.T) {
	blob, err := ioutil.ReadFile("testdata/stureby_highcoin.json")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}
	var genesis core.Genesis
	if err := json.Unmarshal(blob, &genesis); err != nil {
		t.Fatalf("failed parsing genesis: %v", err)
	}
	spec, err := newParityChainSpec("stureby", &genesis, []string{})
	if err != nil {
		t.Fatalf("failed creating chainspec: %v", err)
	}
	enc, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		t.Fatalf("failed encoding chainspec: %v", err)
	}
	expBlob, err := ioutil.ReadFile("testdata/stureby_parity.json")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}
	if !bytes.Equal(expBlob, enc) {
		t.Fatalf("chainspec mismatch")
	}
}
