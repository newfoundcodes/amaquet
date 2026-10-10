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

package protocol

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const (
	Magic               = "AMQT"
	Version        byte = 1
	HeaderSize          = 24
	MaxPayloadSize      = 64 << 20
)

type Kind byte

const (
	KindRequest  Kind = 1
	KindResponse Kind = 2
	KindEvent    Kind = 3
)

type Opcode uint16

const (
	OpCommand     Opcode = 1
	OpHello       Opcode = 2
	OpAuth        Opcode = 3
	OpSubscribe   Opcode = 4
	OpUnsubscribe Opcode = 5
	OpCancel      Opcode = 6
)

type Frame struct {
	Kind      Kind
	Flags     uint16
	Opcode    Opcode
	RequestID uint64
	Payload   []byte
}

func WriteFrame(w io.Writer, f Frame) error {
	if len(f.Payload) > MaxPayloadSize {
		return fmt.Errorf("payload exceeds %d bytes", MaxPayloadSize)
	}
	h := make([]byte, HeaderSize)
	copy(h[0:4], []byte(Magic))
	h[4] = Version
	h[5] = byte(f.Kind)
	binary.BigEndian.PutUint16(h[6:8], f.Flags)
	binary.BigEndian.PutUint16(h[8:10], uint16(f.Opcode))
	binary.BigEndian.PutUint64(h[12:20], f.RequestID)
	binary.BigEndian.PutUint32(h[20:24], uint32(len(f.Payload)))
	if _, e := w.Write(h); e != nil {
		return e
	}
	if len(f.Payload) > 0 {
		_, e := w.Write(f.Payload)
		return e
	}
	return nil
}
func ReadFrame(r io.Reader) (Frame, error) {
	f, release, err := ReadFrameLimitBudget(r, MaxPayloadSize, nil)
	if release != nil {
		release()
	}
	return f, err
}

func ReadFrameLimit(r io.Reader, maxPayload int64) (Frame, error) {
	f, release, err := ReadFrameLimitBudget(r, maxPayload, nil)
	if release != nil {
		release()
	}
	return f, err
}

func ReadFrameLimitBudget(r io.Reader, maxPayload int64, acquire func(int64) (func(), error)) (Frame, func(), error) {
	if maxPayload <= 0 || maxPayload > MaxPayloadSize {
		maxPayload = MaxPayloadSize
	}
	h := make([]byte, HeaderSize)
	if _, e := io.ReadFull(r, h); e != nil {
		return Frame{}, nil, e
	}
	if string(h[0:4]) != Magic {
		return Frame{}, nil, fmt.Errorf("invalid magic")
	}
	if h[4] != Version {
		return Frame{}, nil, fmt.Errorf("unsupported protocol version %d", h[4])
	}
	n := int64(binary.BigEndian.Uint32(h[20:24]))
	if n > maxPayload {
		return Frame{}, nil, fmt.Errorf("payload too large: %d > %d", n, maxPayload)
	}
	var release func()
	if acquire != nil && n > 0 {
		var err error
		release, err = acquire(n)
		if err != nil {
			return Frame{}, nil, err
		}
	}
	p := make([]byte, int(n))
	if _, e := io.ReadFull(r, p); e != nil {
		if release != nil {
			release()
		}
		return Frame{}, nil, e
	}
	return Frame{Kind: Kind(h[5]), Flags: binary.BigEndian.Uint16(h[6:8]), Opcode: Opcode(binary.BigEndian.Uint16(h[8:10])), RequestID: binary.BigEndian.Uint64(h[12:20]), Payload: p}, release, nil
}
func ReadBufferedFrame(r *bufio.Reader) (Frame, error) { return ReadFrame(r) }

type Request struct {
	Command string                     `json:"command"`
	Args    map[string]json.RawMessage `json:"args,omitempty"`
}
type Response struct {
	OK     bool   `json:"ok"`
	Result any    `json:"result,omitempty"`
	Error  *Error `json:"error,omitempty"`
}
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Encode(v any) ([]byte, error)            { return json.Marshal(v) }
func DecodeRequest(b []byte) (Request, error) { var r Request; e := json.Unmarshal(b, &r); return r, e }
