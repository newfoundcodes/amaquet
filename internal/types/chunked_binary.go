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
	"fmt"
	"io"
)

type ChunkedBinary struct {
	Chunks [][]byte
	Size   int64
}

const ChunkedBinaryBlockSize = 1 << 20

func NewChunkedBinaryFromReader(r io.Reader, size int64) (*ChunkedBinary, error) {
	if size < 0 {
		return nil, fmt.Errorf("invalid binary size")
	}
	out := &ChunkedBinary{Size: size}
	remaining := size
	for remaining > 0 {
		n := int64(ChunkedBinaryBlockSize)
		if remaining < n {
			n = remaining
		}
		b := make([]byte, int(n))
		if _, err := io.ReadFull(r, b); err != nil {
			return nil, err
		}
		out.Chunks = append(out.Chunks, b)
		remaining -= n
	}
	return out, nil
}

func (b *ChunkedBinary) ApproxSize() int64 {
	if b == nil {
		return 0
	}
	return b.Size + int64(len(b.Chunks))*32 + 64
}

func (b *ChunkedBinary) ReadRange(offset int64, length int) ([]byte, error) {
	if b == nil || offset < 0 || length < 0 || offset > b.Size {
		return nil, fmt.Errorf("invalid binary range")
	}
	if int64(length) > b.Size-offset {
		length = int(b.Size - offset)
	}
	out := make([]byte, length)
	if length == 0 {
		return out, nil
	}
	pos := 0
	cur := offset
	for pos < length {
		ci := cur / ChunkedBinaryBlockSize
		co := cur % ChunkedBinaryBlockSize
		chunk := b.Chunks[ci]
		n := len(chunk) - int(co)
		if n > length-pos {
			n = length - pos
		}
		copy(out[pos:pos+n], chunk[int(co):int(co)+n])
		pos += n
		cur += int64(n)
	}
	return out, nil
}
