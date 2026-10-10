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
	"math/bits"
	"sort"
	"sync"
)

type Bitmap struct {
	mu    sync.RWMutex
	words []uint64
}

func NewBitmap() *Bitmap { return &Bitmap{} }
func (b *Bitmap) ensure(bit uint64) {
	n := int(bit/64) + 1
	if n > len(b.words) {
		b.words = append(b.words, make([]uint64, n-len(b.words))...)
	}
}
func (b *Bitmap) Set(bit uint64, value bool) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ensure(bit)
	w, m := bit/64, uint64(1)<<(bit%64)
	old := b.words[w]&m != 0
	if value {
		b.words[w] |= m
	} else {
		b.words[w] &^= m
	}
	return old
}
func (b *Bitmap) Get(bit uint64) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	w := int(bit / 64)
	if w >= len(b.words) {
		return false
	}
	return b.words[w]&(uint64(1)<<(bit%64)) != 0
}
func (b *Bitmap) Count() uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var n uint64
	for _, w := range b.words {
		n += uint64(bits.OnesCount64(w))
	}
	return n
}
func (b *Bitmap) Bytes() []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]byte, len(b.words)*8)
	for i, w := range b.words {
		for j := 0; j < 8; j++ {
			out[i*8+j] = byte(w >> (8 * j))
		}
	}
	return out
}

type BitField struct {
	mu   sync.RWMutex
	data []byte
}

func NewBitField(sizeBytes int) *BitField {
	if sizeBytes < 0 {
		sizeBytes = 0
	}
	return &BitField{data: make([]byte, sizeBytes)}
}
func (b *BitField) ensureBits(offset, width uint64) {
	need := int((offset + width + 7) / 8)
	if need > len(b.data) {
		b.data = append(b.data, make([]byte, need-len(b.data))...)
	}
}
func (b *BitField) SetUnsigned(offset, width, value uint64) error {
	if width == 0 || width > 64 {
		return fmt.Errorf("width must be 1..64")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ensureBits(offset, width)
	for i := uint64(0); i < width; i++ {
		pos := offset + i
		mask := byte(1 << (7 - (pos % 8)))
		if value&(uint64(1)<<(width-1-i)) != 0 {
			b.data[pos/8] |= mask
		} else {
			b.data[pos/8] &^= mask
		}
	}
	return nil
}
func (b *BitField) GetUnsigned(offset, width uint64) (uint64, error) {
	if width == 0 || width > 64 {
		return 0, fmt.Errorf("width must be 1..64")
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	if int((offset+width+7)/8) > len(b.data) {
		return 0, nil
	}
	var v uint64
	for i := uint64(0); i < width; i++ {
		pos := offset + i
		v <<= 1
		if b.data[pos/8]&(1<<(7-(pos%8))) != 0 {
			v |= 1
		}
	}
	return v, nil
}

type BitSet = Bitmap

type run16 struct{ start, end uint16 }
type roaringContainer struct {
	kind   uint8 // 0=array, 1=bitmap, 2=run
	array  []uint16
	bitmap []uint64
	runs   []run16
	card   int
}

func (c *roaringContainer) values() []uint16 {
	switch c.kind {
	case 1:
		out := make([]uint16, 0, c.card)
		for i, w := range c.bitmap {
			for w != 0 {
				b := bits.TrailingZeros64(w)
				out = append(out, uint16(i*64+b))
				w &= w - 1
			}
		}
		return out
	case 2:
		out := make([]uint16, 0, c.card)
		for _, r := range c.runs {
			for v := uint32(r.start); v <= uint32(r.end); v++ {
				out = append(out, uint16(v))
			}
		}
		return out
	default:
		return append([]uint16(nil), c.array...)
	}
}
func containerFromValues(xs []uint16) *roaringContainer {
	c := &roaringContainer{kind: 0, array: append([]uint16(nil), xs...), card: len(xs)}
	c.optimize()
	return c
}
func (c *roaringContainer) toArray() {
	if c.kind == 0 {
		return
	}
	c.array = c.values()
	c.bitmap = nil
	c.runs = nil
	c.kind = 0
	c.card = len(c.array)
}
func (c *roaringContainer) optimize() {
	if c.card == 0 {
		return
	}
	xs := c.values()
	runs := make([]run16, 0)
	for i := 0; i < len(xs); {
		j := i
		for j+1 < len(xs) && xs[j+1] == xs[j]+1 {
			j++
		}
		runs = append(runs, run16{xs[i], xs[j]})
		i = j + 1
	}
	if len(runs)*2+4 < len(xs) {
		c.kind = 2
		c.runs = runs
		c.array = nil
		c.bitmap = nil
		return
	}
	if len(xs) > 4096 {
		bm := make([]uint64, 1024)
		for _, v := range xs {
			bm[int(v)/64] |= 1 << (uint(v) % 64)
		}
		c.kind = 1
		c.bitmap = bm
		c.array = nil
		c.runs = nil
		return
	}
	c.kind = 0
	c.array = xs
	c.bitmap = nil
	c.runs = nil
}
func (c *roaringContainer) contains(v uint16) bool {
	switch c.kind {
	case 1:
		return c.bitmap[int(v)/64]&(1<<(uint(v)%64)) != 0
	case 2:
		i := sort.Search(len(c.runs), func(i int) bool { return c.runs[i].end >= v })
		return i < len(c.runs) && c.runs[i].start <= v
	default:
		i := sort.Search(len(c.array), func(i int) bool { return c.array[i] >= v })
		return i < len(c.array) && c.array[i] == v
	}
}
func (c *roaringContainer) add(v uint16) bool {
	if c.contains(v) {
		return false
	}
	c.toArray()
	i := sort.Search(len(c.array), func(i int) bool { return c.array[i] >= v })
	c.array = append(c.array, 0)
	copy(c.array[i+1:], c.array[i:])
	c.array[i] = v
	c.card++
	if c.card == 4097 || c.card%1024 == 0 {
		c.optimize()
	}
	return true
}
func (c *roaringContainer) remove(v uint16) bool {
	if !c.contains(v) {
		return false
	}
	c.toArray()
	i := sort.Search(len(c.array), func(i int) bool { return c.array[i] >= v })
	c.array = append(c.array[:i], c.array[i+1:]...)
	c.card--
	if c.card > 0 && (c.card == 2048 || c.card%1024 == 0) {
		c.optimize()
	}
	return true
}

type RoaringBitmap struct {
	mu         sync.RWMutex
	containers map[uint16]*roaringContainer
}

func NewRoaringBitmap() *RoaringBitmap {
	return &RoaringBitmap{containers: map[uint16]*roaringContainer{}}
}
func (r *RoaringBitmap) Add(v uint32) bool {
	hi, lo := uint16(v>>16), uint16(v)
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.containers[hi]
	if c == nil {
		c = &roaringContainer{}
		r.containers[hi] = c
	}
	return c.add(lo)
}
func (r *RoaringBitmap) Remove(v uint32) bool {
	hi, lo := uint16(v>>16), uint16(v)
	r.mu.Lock()
	defer r.mu.Unlock()
	c := r.containers[hi]
	if c == nil {
		return false
	}
	ok := c.remove(lo)
	if c.card == 0 {
		delete(r.containers, hi)
	}
	return ok
}
func (r *RoaringBitmap) Contains(v uint32) bool {
	hi, lo := uint16(v>>16), uint16(v)
	r.mu.RLock()
	defer r.mu.RUnlock()
	c := r.containers[hi]
	return c != nil && c.contains(lo)
}
func (r *RoaringBitmap) Cardinality() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var n uint64
	for _, c := range r.containers {
		n += uint64(c.card)
	}
	return n
}
func (r *RoaringBitmap) Values() []uint32 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	his := make([]int, 0, len(r.containers))
	for hi := range r.containers {
		his = append(his, int(hi))
	}
	sort.Ints(his)
	out := make([]uint32, 0)
	for _, h := range his {
		for _, lo := range r.containers[uint16(h)].values() {
			out = append(out, uint32(h)<<16|uint32(lo))
		}
	}
	return out
}
func (r *RoaringBitmap) Union(o *RoaringBitmap) *RoaringBitmap {
	n := NewRoaringBitmap()
	for _, v := range r.Values() {
		n.Add(v)
	}
	for _, v := range o.Values() {
		n.Add(v)
	}
	return n
}
func (r *RoaringBitmap) Intersect(o *RoaringBitmap) *RoaringBitmap {
	n := NewRoaringBitmap()
	a, b := r.Values(), o.Values()
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			n.Add(a[i])
			i++
			j++
		} else if a[i] < b[j] {
			i++
		} else {
			j++
		}
	}
	return n
}
