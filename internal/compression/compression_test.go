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
	"strings"
	"testing"
)

func TestRoundTrips(t *testing.T) {
	samples := [][]byte{
		{}, []byte("abc"), []byte(strings.Repeat("Amaquet compression test ", 1000)),
		bytes.Repeat([]byte{0}, 10000), bytes.Repeat([]byte("abcd"), 5000),
	}
	deterministic := make([]byte, 8192)

	var x uint32 = 0x12345678
	for i := range deterministic {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5

		deterministic[i] = byte(x)
	}

	samples = append(samples, deterministic)
	for _, s := range samples {
		for _, alg := range []string{AlgorithmLZ4, AlgorithmRLE, AlgorithmDeflateFast} {
			var enc []byte
			var err error

			switch alg {
			case AlgorithmLZ4:
				enc, err = compressLZ4Block(s)
			case AlgorithmRLE:
				enc, err = compressRLE(s)
			case AlgorithmDeflateFast:
				enc, err = compressDeflateFast(s)
			}

			if err != nil {
				t.Fatalf("%s compress: %v", alg, err)
			}

			p := Payload{Algorithm: alg, OriginalSize: len(s), Data: enc}
			m, _ := NewManager(DefaultConfig())
			got, err := m.Decompress(p)

			if err != nil {
				t.Fatalf("%s decompress: %v", alg, err)
			}

			if !bytes.Equal(got, s) {
				t.Fatalf("%s round-trip mismatch for %d bytes", alg, len(s))
			}
		}
	}
}

func TestManagerThreshold(t *testing.T) {
	m, err := NewManager(Config{Enabled: true, Algorithm: AlgorithmAuto, MinBytes: 64, MinSavingsPercent: 10})
	if err != nil {
		t.Fatal(err)
	}

	src := []byte(strings.Repeat("aaaaaaaaaaaaaaaaJSON", 1000))
	p, ok, err := m.Compress(src)

	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("expected compressible payload")
	}

	got, err := m.Decompress(p)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(got, src) {
		t.Fatal("round trip mismatch")
	}
}

func TestRLELiteralBoundaryRegression(t *testing.T) {
	for _, n := range []int{127, 128, 129, 255, 256, 257} {
		src := make([]byte, n)
		for i := range src {
			src[i] = byte((i*73 + 19) % 251)
			if i > 0 && src[i] == src[i-1] {
				src[i]++
			}
		}

		enc, err := compressRLE(src)
		if err != nil {
			t.Fatalf("n=%d compress: %v", n, err)
		}

		dec, err := decompressRLE(enc, len(src))
		if err != nil {
			t.Fatalf("n=%d decompress: %v", n, err)
		}

		if !bytes.Equal(src, dec) {
			t.Fatalf("n=%d round trip mismatch", n)
		}
	}
}
