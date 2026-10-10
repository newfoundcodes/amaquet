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
	"bytes"
	"testing"
)

func FuzzReadFrame(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("AMQT\x01\x01"))
	f.Add(bytes.Repeat([]byte{0xff}, HeaderSize+8))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}
		_, _ = ReadFrameLimit(bytes.NewReader(data), 1<<20)
	})
}

func FuzzFrameRoundTrip(f *testing.F) {
	f.Add(uint8(KindRequest), uint16(0), uint16(OpCommand), uint64(1), []byte(`{"command":"PING"}`))
	f.Add(uint8(KindEvent), uint16(7), uint16(OpSubscribe), uint64(999), []byte("hello"))
	f.Fuzz(func(t *testing.T, kind uint8, flags uint16, opcode uint16, requestID uint64, payload []byte) {
		if len(payload) > 1<<16 {
			return
		}
		in := Frame{Kind: Kind(kind), Flags: flags, Opcode: Opcode(opcode), RequestID: requestID, Payload: payload}
		var b bytes.Buffer
		if err := WriteFrame(&b, in); err != nil {
			t.Fatalf("write: %v", err)
		}
		out, err := ReadFrame(&b)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if out.Kind != in.Kind || out.Flags != in.Flags || out.Opcode != in.Opcode || out.RequestID != in.RequestID || !bytes.Equal(out.Payload, in.Payload) {
			t.Fatalf("round trip mismatch: in=%+v out=%+v", in, out)
		}
	})
}
