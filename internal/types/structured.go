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

package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/newfoundcodes/amaquet/internal/core"
	"math"
	"sync"
)

func decodeJSONPreserveNumbers(raw []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

type JSONDocument struct {
	mu    sync.RWMutex
	value any
}

func NewJSONDocument(raw []byte) (*JSONDocument, error) {
	v, err := decodeJSONPreserveNumbers(raw)
	if err != nil {
		return nil, err
	}
	return &JSONDocument{value: v}, nil
}
func (j *JSONDocument) Marshal() ([]byte, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return json.Marshal(j.value)
}
func (j *JSONDocument) Value() any {
	j.mu.RLock()
	defer j.mu.RUnlock()
	b, _ := json.Marshal(j.value)
	v, _ := decodeJSONPreserveNumbers(b)
	return v
}
func (j *JSONDocument) Set(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c, err := decodeJSONPreserveNumbers(b)
	if err != nil {
		return err
	}
	j.mu.Lock()
	j.value = c
	j.mu.Unlock()
	return nil
}

type MessagePack struct{ Data []byte }

func NewMessagePack(data []byte) MessagePack { return MessagePack{Data: append([]byte(nil), data...)} }

type CBOR struct{ Data []byte }

func NewCBOR(data []byte) CBOR { return CBOR{Data: append([]byte(nil), data...)} }

type RecordField struct {
	Name     string        `json:"name"`
	Type     core.DataType `json:"type"`
	Nullable bool          `json:"nullable"`
}
type Record struct {
	mu     sync.RWMutex
	Schema []RecordField
	Fields map[string]core.Value
}

func NewRecord(schema []RecordField) *Record {
	return &Record{Schema: append([]RecordField(nil), schema...), Fields: map[string]core.Value{}}
}
func (r *Record) Set(name string, v core.Value) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.Schema {
		if f.Name == name {
			if v.Type == core.TypeNull && f.Nullable {
				r.Fields[name] = v
				return nil
			}
			if v.Type != f.Type {
				return fmt.Errorf("field %s expects %s", name, f.Type)
			}
			r.Fields[name] = v
			return nil
		}
	}
	return fmt.Errorf("unknown field %q", name)
}
func (r *Record) Get(name string) (core.Value, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.Fields[name]
	return v, ok
}

type NumericDType string

const (
	DTypeFloat64 NumericDType = "float64"
	DTypeFloat32 NumericDType = "float32"
	DTypeInt64   NumericDType = "int64"
	DTypeInt32   NumericDType = "int32"
)

type Tensor struct {
	mu    sync.RWMutex
	Shape []int        `json:"shape"`
	DType NumericDType `json:"dtype"`
	Data  []float64    `json:"data"`
}

func NewTensor(shape []int, dtype NumericDType, data []float64) (*Tensor, error) {
	n := 1
	for _, d := range shape {
		if d <= 0 {
			return nil, fmt.Errorf("invalid dimension")
		}
		if n > math.MaxInt/d {
			return nil, fmt.Errorf("tensor too large")
		}
		n *= d
	}
	if n != len(data) {
		return nil, fmt.Errorf("shape requires %d values, got %d", n, len(data))
	}
	return &Tensor{Shape: append([]int(nil), shape...), DType: dtype, Data: append([]float64(nil), data...)}, nil
}
func (t *Tensor) offset(indices []int) (int, error) {
	if len(indices) != len(t.Shape) {
		return 0, fmt.Errorf("rank mismatch")
	}
	off := 0
	stride := 1
	for i := len(t.Shape) - 1; i >= 0; i-- {
		if indices[i] < 0 || indices[i] >= t.Shape[i] {
			return 0, fmt.Errorf("index out of range")
		}
		off += indices[i] * stride
		stride *= t.Shape[i]
	}
	return off, nil
}
func (t *Tensor) Get(indices ...int) (float64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	o, e := t.offset(indices)
	if e != nil {
		return 0, e
	}
	return t.Data[o], nil
}
func (t *Tensor) Set(value float64, indices ...int) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	o, e := t.offset(indices)
	if e != nil {
		return e
	}
	t.Data[o] = value
	return nil
}
func (t *Tensor) Size() int { t.mu.RLock(); defer t.mu.RUnlock(); return len(t.Data) }

type Matrix = Tensor

func NewMatrix(rows, cols int, dtype NumericDType, data []float64) (*Matrix, error) {
	return NewTensor([]int{rows, cols}, dtype, data)
}
