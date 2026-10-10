/*
 * Amaquet - A Newfoundcodes project.
 *
 * Copyright (C) 2026 Jonathan Eldy Baldivicio
 *
 * Author: Jonathan Eldy Baldivicio
 * Contact: jonathaneldy.baldivicio@newfoundcodes.com
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

package server

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/newfoundcodes/amaquet/internal/compression"
	"github.com/newfoundcodes/amaquet/internal/core"
	nt "github.com/newfoundcodes/amaquet/internal/types"
)

func compressibleBytes(v core.Value) ([]byte, bool, error) {
	switch x := v.Data.(type) {
	case nt.BinaryString:
		return append([]byte(nil), []byte(x)...), true, nil
	case nt.UTF8String:
		return []byte(string(x)), true, nil
	case nt.Decimal:
		return []byte(x.String()), true, nil
	case nt.BigInteger:
		return []byte(x.String()), true, nil
	case *nt.JSONDocument:
		b, err := x.Marshal()
		return b, true, err
	case nt.MessagePack:
		return append([]byte(nil), x.Data...), true, nil
	case nt.CBOR:
		return append([]byte(nil), x.Data...), true, nil
	case nt.DenseVector:
		b := make([]byte, 4*len(x))
		for i, n := range x {
			binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(n))
		}
		return b, true, nil
	case nt.SparseVector:
		keys := make([]int, 0, len(x.Values))
		for k := range x.Values {
			keys = append(keys, k)
		}
		sort.Ints(keys)
		b := make([]byte, 8+8*len(keys))
		binary.LittleEndian.PutUint32(b[0:4], uint32(x.Dim))
		binary.LittleEndian.PutUint32(b[4:8], uint32(len(keys)))
		p := 8
		for _, k := range keys {
			binary.LittleEndian.PutUint32(b[p:p+4], uint32(k))
			binary.LittleEndian.PutUint32(b[p+4:p+8], math.Float32bits(x.Values[k]))
			p += 8
		}
		return b, true, nil
	case nt.QuantizedVector:
		b := make([]byte, 5+len(x.Data))
		binary.LittleEndian.PutUint32(b[0:4], math.Float32bits(x.Scale))
		b[4] = byte(x.ZeroPoint)
		for i, n := range x.Data {
			b[5+i] = byte(n)
		}
		return b, true, nil
	case *nt.Tensor:
		snap := x.Snapshot().(map[string]any)
		shape := snap["shape"].([]int)
		dtype := string(snap["dtype"].(nt.NumericDType))
		data := snap["data"].([]float64)
		if len(dtype) > 255 || len(shape) > 255 {
			return nil, false, fmt.Errorf("tensor metadata too large")
		}
		b := make([]byte, 2+len(dtype)+4*len(shape)+8*len(data))
		b[0] = byte(len(dtype))
		b[1] = byte(len(shape))
		copy(b[2:], dtype)
		p := 2 + len(dtype)
		for _, n := range shape {
			binary.LittleEndian.PutUint32(b[p:p+4], uint32(n))
			p += 4
		}
		for _, n := range data {
			binary.LittleEndian.PutUint64(b[p:p+8], math.Float64bits(n))
			p += 8
		}
		return b, true, nil
	default:
		return nil, false, nil
	}
}

func decodeCompressedValue(t core.DataType, raw []byte) (core.Value, error) {
	var data any
	switch t {
	case core.TypeBinaryString:
		data = nt.BinaryString(append([]byte(nil), raw...))
	case core.TypeUTF8String:
		s, err := nt.NewUTF8String(string(raw))
		if err != nil {
			return core.Value{}, err
		}
		data = s
	case core.TypeDecimal, core.TypeBigDecimal:
		x, err := nt.ParseDecimal(string(raw))
		if err != nil {
			return core.Value{}, err
		}
		data = x
	case core.TypeBigInteger:
		x, err := nt.ParseBigInteger(string(raw))
		if err != nil {
			return core.Value{}, err
		}
		data = x
	case core.TypeJSON:
		x, err := nt.NewJSONDocument(raw)
		if err != nil {
			return core.Value{}, err
		}
		data = x
	case core.TypeMessagePack:
		data = nt.NewMessagePack(raw)
	case core.TypeCBOR:
		data = nt.NewCBOR(raw)
	case core.TypeDenseVector:
		if len(raw)%4 != 0 {
			return core.Value{}, fmt.Errorf("corrupt compressed dense vector")
		}
		x := make(nt.DenseVector, len(raw)/4)
		for i := range x {
			x[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		}
		data = x
	case core.TypeSparseVector:
		if len(raw) < 8 {
			return core.Value{}, fmt.Errorf("corrupt compressed sparse vector")
		}
		dim, n := int(binary.LittleEndian.Uint32(raw[0:4])), int(binary.LittleEndian.Uint32(raw[4:8]))
		if len(raw) != 8+n*8 {
			return core.Value{}, fmt.Errorf("corrupt compressed sparse vector length")
		}
		vals := make(map[int]float32, n)
		p := 8
		for i := 0; i < n; i++ {
			k := int(binary.LittleEndian.Uint32(raw[p : p+4]))
			f := math.Float32frombits(binary.LittleEndian.Uint32(raw[p+4 : p+8]))
			vals[k] = f
			p += 8
		}
		x, err := nt.NewSparseVector(dim, vals)
		if err != nil {
			return core.Value{}, err
		}
		data = x
	case core.TypeQuantizedVector:
		if len(raw) < 5 {
			return core.Value{}, fmt.Errorf("corrupt compressed quantized vector")
		}
		x := nt.QuantizedVector{Scale: math.Float32frombits(binary.LittleEndian.Uint32(raw[0:4])), ZeroPoint: int8(raw[4]), Data: make([]int8, len(raw)-5)}
		for i, b := range raw[5:] {
			x.Data[i] = int8(b)
		}
		data = x
	case core.TypeMatrix, core.TypeTensor:
		if len(raw) < 2 {
			return core.Value{}, fmt.Errorf("corrupt compressed tensor")
		}
		dn, rank := int(raw[0]), int(raw[1])
		p := 2
		if p+dn+4*rank > len(raw) {
			return core.Value{}, fmt.Errorf("corrupt compressed tensor metadata")
		}
		dtype := nt.NumericDType(string(raw[p : p+dn]))
		p += dn
		shape := make([]int, rank)
		count := 1
		for i := range shape {
			shape[i] = int(binary.LittleEndian.Uint32(raw[p : p+4]))
			p += 4
			count *= shape[i]
		}
		if p+8*count != len(raw) {
			return core.Value{}, fmt.Errorf("corrupt compressed tensor length")
		}
		vals := make([]float64, count)
		for i := range vals {
			vals[i] = math.Float64frombits(binary.LittleEndian.Uint64(raw[p : p+8]))
			p += 8
		}
		x, err := nt.NewTensor(shape, dtype, vals)
		if err != nil {
			return core.Value{}, err
		}
		data = x
	default:
		return core.Value{}, fmt.Errorf("type %s does not support compressed storage", t)
	}
	return core.Value{Type: t, Data: data}, nil
}

func (d *Dispatcher) prepareValue(v core.Value) (core.Value, error) {
	if d.compressor == nil {
		return v, nil
	}
	raw, ok, err := compressibleBytes(v)
	if err != nil || !ok {
		return v, err
	}
	payload, compressed, err := d.compressor.Compress(raw)
	if err != nil || !compressed {
		return v, err
	}
	v.Data = payload
	return v, nil
}

func (d *Dispatcher) materializeValue(v core.Value) (core.Value, error) {
	p, ok := v.Data.(compression.Payload)
	if !ok {
		return v, nil
	}
	if d.compressor == nil {
		return core.Value{}, fmt.Errorf("compressed value encountered without compression manager")
	}
	raw, err := d.compressor.Decompress(p)
	if err != nil {
		return core.Value{}, err
	}
	return decodeCompressedValue(v.Type, raw)
}

func compressionPayloadInfo(v core.Value) (map[string]any, bool) {
	p, ok := v.Data.(compression.Payload)
	if !ok {
		return nil, false
	}
	savings := p.OriginalSize - len(p.Data)
	percent := float64(0)
	if p.OriginalSize > 0 {
		percent = 100 * float64(savings) / float64(p.OriginalSize)
	}
	return map[string]any{
		"compressed": true, "algorithm": p.Algorithm, "original_bytes": p.OriginalSize,
		"stored_bytes": len(p.Data), "savings_bytes": savings, "savings_percent": percent,
	}, true
}

func MarshalCompressedDebug(v core.Value) ([]byte, error) { return json.Marshal(v.Data) }
