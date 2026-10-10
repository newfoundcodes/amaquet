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

package compression

import (
	"bytes"
	"testing"
)

func FuzzCompressionRoundTrip(f *testing.F) {
	f.Add([]byte("hello hello hello hello"))
	f.Add(bytes.Repeat([]byte{0}, 4096))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, src []byte) {
		if len(src) > 1<<20 {
			return
		}

		for _, alg := range []string{AlgorithmAuto, AlgorithmLZ4, AlgorithmRLE, AlgorithmDeflateFast} {
			m, err := NewManager(Config{Enabled: true, Algorithm: alg, MinBytes: 0, MinSavingsPercent: 0})
			if err != nil {
				t.Fatal(err)
			}

			p, ok, err := m.Compress(src)
			if err != nil {
				t.Fatalf("%s compress: %v", alg, err)
			}

			if !ok {
				continue
			}

			got, err := m.Decompress(p)
			if err != nil {
				t.Fatalf("%s decompress: %v", alg, err)
			}

			if !bytes.Equal(got, src) {
				t.Fatalf("%s round trip mismatch", alg)
			}
		}
	})
}

func FuzzMalformedCompressedPayload(f *testing.F) {
	f.Add("lz4", 16, []byte{0xff})
	f.Add("rle", 10, []byte{0x80})
	f.Add("deflate-fast", 10, []byte("not-deflate"))

	f.Fuzz(func(t *testing.T, alg string, original int, data []byte) {
		if len(data) > 1<<20 {
			return
		}

		if original < 0 {
			original = -original
		}

		original %= 1 << 20
		if alg != AlgorithmLZ4 && alg != AlgorithmRLE && alg != AlgorithmDeflateFast {
			alg = AlgorithmLZ4
		}

		m, _ := NewManager(DefaultConfig())
		_, _ = m.Decompress(Payload{Algorithm: alg, OriginalSize: original, Data: data})
	})
}
