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

package vm

import (
	"encoding/json"
	"io"
	"math/big"
	"time"

	"github.com/420integrated/go-highcoin/common"
	"github.com/420integrated/go-highcoin/common/math"
)

type JSONLogger struct {
	encoder *json.Encoder
	cfg     *LogConfig
}

// NewJSONLogger creates a new EVM tracer that prints execution steps as JSON objects
// into the provided stream.
func NewJSONLogger(cfg *LogConfig, writer io.Writer) *JSONLogger {
	l := &JSONLogger{json.NewEncoder(writer), cfg}
	if l.cfg == nil {
		l.cfg = &LogConfig{}
	}
	return l
}

func (l *JSONLogger) CaptureStart(from common.Address, to common.Address, create bool, input []byte, smoke uint64, value *big.Int) error {
	return nil
}

// CaptureState outputs state information on the logger.
func (l *JSONLogger) CaptureState(env *EVM, pc uint64, op OpCode, smoke, cost uint64, memory *Memory, stack *Stack, rStack *ReturnStack, rData []byte, contract *Contract, depth int, err error) error {
	log := StructLog{
		Pc:            pc,
		Op:            op,
		Smoke:           smoke,
		SmokeCost:       cost,
		MemorySize:    memory.Len(),
		Storage:       nil,
		Depth:         depth,
		RefundCounter: env.StateDB.GetRefund(),
		Err:           err,
	}
	if !l.cfg.DisableMemory {
		log.Memory = memory.Data()
	}
	if !l.cfg.DisableStack {
		//TODO(@holiman) improve this
		logstack := make([]*big.Int, len(stack.Data()))
		for i, item := range stack.Data() {
			logstack[i] = item.ToBig()
		}
		log.Stack = logstack
		log.ReturnStack = rStack.data
	}
	if !l.cfg.DisableReturnData {
		log.ReturnData = rData
	}
	return l.encoder.Encode(log)
}

// CaptureFault outputs state information on the logger.
func (l *JSONLogger) CaptureFault(env *EVM, pc uint64, op OpCode, smoke, cost uint64, memory *Memory, stack *Stack, rStack *ReturnStack, contract *Contract, depth int, err error) error {
	return nil
}

// CaptureEnd is triggered at end of execution.
func (l *JSONLogger) CaptureEnd(output []byte, smokeUsed uint64, t time.Duration, err error) error {
	type endLog struct {
		Output  string              `json:"output"`
		SmokeUsed math.HexOrDecimal64 `json:"smokeUsed"`
		Time    time.Duration       `json:"time"`
		Err     string              `json:"error,omitempty"`
	}
	if err != nil {
		return l.encoder.Encode(endLog{common.Bytes2Hex(output), math.HexOrDecimal64(smokeUsed), t, err.Error()})
	}
	return l.encoder.Encode(endLog{common.Bytes2Hex(output), math.HexOrDecimal64(smokeUsed), t, ""})
}
