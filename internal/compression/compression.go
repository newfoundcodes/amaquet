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
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io"
)

type Payload struct {
	Algorithm    string `json:"algorithm"`
	OriginalSize int    `json:"original_size"`
	Data         []byte `json:"data"`
}

type Config struct {
	Enabled           bool    `json:"enabled"`
	Algorithm         string  `json:"algorithm"`
	MinBytes          int     `json:"min_bytes"`
	MinSavingsPercent float64 `json:"min_savings_percent"`
}

type Manager struct{ cfg Config }

const (
	AlgorithmAuto        = "auto"
	AlgorithmLZ4         = "lz4"
	AlgorithmRLE         = "rle"
	AlgorithmDeflateFast = "deflate-fast"
	AlgorithmNone        = "none"
)

func DefaultConfig() Config {
	return Config{
		Enabled:           true,
		Algorithm:         AlgorithmAuto,
		MinBytes:          1024,
		MinSavingsPercent: 8,
	}
}

func ValidateConfig(c Config) error {
	switch c.Algorithm {
	case AlgorithmAuto, AlgorithmLZ4, AlgorithmRLE, AlgorithmDeflateFast, AlgorithmNone:
	default:
		return fmt.Errorf("unsupported compression algorithm %q", c.Algorithm)
	}

	if c.MinBytes < 0 {
		return fmt.Errorf("compression.min_bytes must be non-negative")
	}

	if c.MinSavingsPercent < 0 || c.MinSavingsPercent >= 100 {
		return fmt.Errorf("compression.min_savings_percent must be >= 0 and < 100")
	}

	return nil
}

func (p Payload) StoredSize() int {
	return len(p.Data)
}

func (p Payload) SavingsBytes() int {
	return p.OriginalSize - len(p.Data)
}

func (p Payload) Ratio() float64 {
	if p.OriginalSize == 0 {
		return 1
	}

	return float64(len(p.Data)) / float64(p.OriginalSize)
}

func NewManager(cfg Config) (*Manager, error) {
	if cfg.Algorithm == "" {
		cfg.Algorithm = AlgorithmAuto
	}

	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return &Manager{cfg: cfg}, nil
}

func (m *Manager) Config() Config {
	return m.cfg
}

func (m *Manager) Compress(src []byte) (Payload, bool, error) {
	if m == nil || !m.cfg.Enabled || m.cfg.Algorithm == AlgorithmNone || len(src) < m.cfg.MinBytes {
		return Payload{}, false, nil
	}

	alg := m.cfg.Algorithm

	var best []byte
	bestAlg := ""

	try := func(name string, fn func([]byte) ([]byte, error)) error {
		out, err := fn(src)
		if err != nil {
			return err
		}

		if len(out) < len(best) || best == nil {
			best = out
			bestAlg = name
		}

		return nil
	}

	switch alg {
	case AlgorithmAuto:
		if err := try(AlgorithmRLE, compressRLE); err != nil {
			return Payload{}, false, err
		}

		if err := try(AlgorithmLZ4, compressLZ4Block); err != nil {
			return Payload{}, false, err
		}
	case AlgorithmLZ4:
		if err := try(AlgorithmLZ4, compressLZ4Block); err != nil {
			return Payload{}, false, err
		}
	case AlgorithmRLE:
		if err := try(AlgorithmRLE, compressRLE); err != nil {
			return Payload{}, false, err
		}
	case AlgorithmDeflateFast:
		if err := try(AlgorithmDeflateFast, compressDeflateFast); err != nil {
			return Payload{}, false, err
		}
	}

	if best == nil || len(best) >= len(src) {
		return Payload{}, false, nil
	}

	savings := 100 * (1 - float64(len(best))/float64(len(src)))
	if savings < m.cfg.MinSavingsPercent {
		return Payload{}, false, nil
	}

	cp := append([]byte(nil), best...)
	return Payload{
		Algorithm:    bestAlg,
		OriginalSize: len(src),
		Data:         cp,
	}, true, nil
}

func (m *Manager) Decompress(p Payload) ([]byte, error) {
	switch p.Algorithm {
	case AlgorithmLZ4:
		return decompressLZ4Block(p.Data, p.OriginalSize)
	case AlgorithmRLE:
		return decompressRLE(p.Data, p.OriginalSize)
	case AlgorithmDeflateFast:
		return decompressDeflate(p.Data, p.OriginalSize)
	default:
		return nil, fmt.Errorf("unknown compression algorithm %q", p.Algorithm)
	}
}

func compressDeflateFast(src []byte) ([]byte, error) {
	var b bytes.Buffer
	w, err := flate.NewWriter(&b, flate.BestSpeed)

	if err != nil {
		return nil, err
	}

	if _, err = w.Write(src); err != nil {
		_ = w.Close()
		return nil, err
	}

	if err = w.Close(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func decompressDeflate(src []byte, original int) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(src))
	defer r.Close()

	var b bytes.Buffer
	if original > 0 {
		b.Grow(original)
	}

	if _, err := io.Copy(&b, r); err != nil {
		return nil, err
	}

	out := b.Bytes()
	if original >= 0 && len(out) != original {
		return nil, fmt.Errorf("deflate length mismatch: got %d want %d", len(out), original)
	}

	return out, nil
}

func compressRLE(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}

	out := make([]byte, 0, len(src)/2)
	for i := 0; i < len(src); {
		run := 1
		for i+run < len(src) && src[i+run] == src[i] && run < 130 {
			run++
		}

		if run >= 3 {
			out = append(out, 0x80|byte(run-3), src[i])
			i += run
			continue
		}

		start := i
		i += run

		for i < len(src) && i-start < 128 {
			r := 1
			for i+r < len(src) && src[i+r] == src[i] && r < 130 {
				r++
			}

			if r >= 3 {
				break
			}

			remaining := 128 - (i - start)
			if r > remaining {
				r = remaining
			}

			i += r
		}

		n := i - start
		out = append(out, byte(n-1))
		out = append(out, src[start:i]...)
	}

	return out, nil
}

func decompressRLE(src []byte, original int) ([]byte, error) {
	out := make([]byte, 0, original)
	for i := 0; i < len(src); {
		c := src[i]
		i++

		if c&0x80 == 0 {
			n := int(c) + 1
			if i+n > len(src) {
				return nil, io.ErrUnexpectedEOF
			}

			out = append(out, src[i:i+n]...)
			i += n
		} else {
			n := int(c&0x7f) + 3
			if i >= len(src) {
				return nil, io.ErrUnexpectedEOF
			}

			v := src[i]
			i++

			for j := 0; j < n; j++ {
				out = append(out, v)
			}
		}

		if original >= 0 && len(out) > original {
			return nil, fmt.Errorf("rle output exceeds expected length")
		}
	}

	if original >= 0 && len(out) != original {
		return nil, fmt.Errorf("rle length mismatch: got %d want %d", len(out), original)
	}

	return out, nil
}

func compressLZ4Block(src []byte) ([]byte, error) {
	const hashBits = 16
	const hashSize = 1 << hashBits

	if len(src) < 4 {
		out := []byte{byte(len(src) << 4)}
		out = append(out, src...)
		return out, nil
	}

	table := make([]int, hashSize)
	for i := range table {
		table[i] = -1
	}

	out := make([]byte, 0, len(src))
	anchor := 0

	for i := 0; i+4 <= len(src); {
		seq := binary.LittleEndian.Uint32(src[i : i+4])
		h := int((seq * uint32(2654435761)) >> (32 - hashBits))
		ref := table[h]

		table[h] = i
		if ref < 0 || i-ref > 65535 || ref+4 > len(src) || !bytes.Equal(src[ref:ref+4], src[i:i+4]) {
			i++
			continue
		}

		litLen := i - anchor
		matchLen := 4

		for i+matchLen < len(src) && ref+matchLen < i && src[ref+matchLen] == src[i+matchLen] {
			matchLen++
		}

		for i+matchLen < len(src) && src[ref+(matchLen%(i-ref))] == src[i+matchLen] {
			matchLen++
		}

		tokenPos := len(out)
		out = append(out, 0)
		token := byte(0)

		if litLen < 15 {
			token |= byte(litLen << 4)
		} else {
			token |= 0xF0
			n := litLen - 15

			for n >= 255 {
				out = append(out, 255)
				n -= 255
			}

			out = append(out, byte(n))
		}

		out = append(out, src[anchor:i]...)
		off := i - ref

		out = append(out, byte(off), byte(off>>8))
		ml := matchLen - 4

		if ml < 15 {
			token |= byte(ml)
		} else {
			token |= 0x0F
			n := ml - 15

			for n >= 255 {
				out = append(out, 255)
				n -= 255
			}

			out = append(out, byte(n))
		}

		out[tokenPos] = token
		i += matchLen
		anchor = i

		for p := i - 2; p < i && p >= 0 && p+4 <= len(src); p++ {
			s := binary.LittleEndian.Uint32(src[p : p+4])
			hh := int((s * uint32(2654435761)) >> (32 - hashBits))
			table[hh] = p
		}
	}

	litLen := len(src) - anchor
	tokenPos := len(out)
	out = append(out, 0)

	token := byte(0)
	if litLen < 15 {
		token = byte(litLen << 4)
	} else {
		token = 0xF0
		n := litLen - 15

		for n >= 255 {
			out = append(out, 255)
			n -= 255
		}

		out = append(out, byte(n))
	}

	out = append(out, src[anchor:]...)
	out[tokenPos] = token

	return out, nil
}

func decompressLZ4Block(src []byte, original int) ([]byte, error) {
	out := make([]byte, 0, original)
	for i := 0; i < len(src); {
		token := src[i]
		i++

		litLen := int(token >> 4)
		if litLen == 15 {
			for {
				if i >= len(src) {
					return nil, io.ErrUnexpectedEOF
				}

				x := int(src[i])
				i++
				litLen += x

				if x != 255 {
					break
				}
			}
		}

		if i+litLen > len(src) {
			return nil, io.ErrUnexpectedEOF
		}

		out = append(out, src[i:i+litLen]...)
		i += litLen

		if i == len(src) {
			break
		}

		if i+2 > len(src) {
			return nil, io.ErrUnexpectedEOF
		}

		off := int(src[i]) | int(src[i+1])<<8
		i += 2

		if off <= 0 || off > len(out) {
			return nil, fmt.Errorf("invalid lz4 offset %d", off)
		}

		matchLen := int(token&0x0f) + 4
		if token&0x0f == 15 {
			for {
				if i >= len(src) {
					return nil, io.ErrUnexpectedEOF
				}

				x := int(src[i])
				i++

				matchLen += x
				if x != 255 {
					break
				}
			}
		}

		start := len(out) - off
		for j := 0; j < matchLen; j++ {
			out = append(out, out[start+j])
		}

		if original >= 0 && len(out) > original {
			return nil, fmt.Errorf("lz4 output exceeds expected length")
		}
	}

	if original >= 0 && len(out) != original {
		return nil, fmt.Errorf("lz4 length mismatch: got %d want %d", len(out), original)
	}

	return out, nil
}
