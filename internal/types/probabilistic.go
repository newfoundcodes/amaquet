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
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"sync"
)

func hash64(seed uint64, data []byte) uint64 {
	h := sha256.New()
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], seed)
	_, _ = h.Write(b[:])
	_, _ = h.Write(data)
	sum := h.Sum(nil)
	return binary.BigEndian.Uint64(sum[:8])
}

type HyperLogLog struct {
	mu        sync.Mutex
	Precision uint8
	registers []uint8
}

func NewHyperLogLog(precision uint8) (*HyperLogLog, error) {
	if precision < 4 || precision > 18 {
		return nil, fmt.Errorf("precision must be 4..18")
	}
	return &HyperLogLog{Precision: precision, registers: make([]uint8, 1<<precision)}, nil
}
func (h *HyperLogLog) Add(data []byte) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	x := hash64(0, data)
	idx := x >> (64 - h.Precision)
	w := x << h.Precision
	rank := uint8(1)
	for rank <= 64-h.Precision && w&(uint64(1)<<63) == 0 {
		rank++
		w <<= 1
	}
	if rank > h.registers[idx] {
		h.registers[idx] = rank
		return true
	}
	return false
}
func (h *HyperLogLog) Count() uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := float64(len(h.registers))
	alpha := 0.7213 / (1 + 1.079/m)
	sum := 0.0
	zeros := 0
	for _, r := range h.registers {
		sum += math.Ldexp(1, -int(r))
		if r == 0 {
			zeros++
		}
	}
	estimate := alpha * m * m / sum
	if estimate <= 2.5*m && zeros > 0 {
		estimate = m * math.Log(m/float64(zeros))
	}
	if estimate < 0 {
		return 0
	}
	return uint64(estimate + 0.5)
}

type BloomFilter struct {
	mu   sync.RWMutex
	bits []uint64
	K    uint32
	M    uint64
}

func NewBloomFilter(expected uint64, falsePositiveRate float64) (*BloomFilter, error) {
	if expected == 0 || falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		return nil, fmt.Errorf("invalid bloom parameters")
	}
	m := uint64(math.Ceil(-float64(expected) * math.Log(falsePositiveRate) / (math.Ln2 * math.Ln2)))
	k := uint32(math.Max(1, math.Round(float64(m)/float64(expected)*math.Ln2)))
	return &BloomFilter{bits: make([]uint64, (m+63)/64), K: k, M: m}, nil
}
func (b *BloomFilter) locations(data []byte) []uint64 {
	h1 := hash64(1, data)
	h2 := hash64(2, data) | 1
	out := make([]uint64, b.K)
	for i := uint32(0); i < b.K; i++ {
		out[i] = (h1 + uint64(i)*h2) % b.M
	}
	return out
}
func (b *BloomFilter) Add(data []byte) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	changed := false
	for _, p := range b.locations(data) {
		w, m := p/64, uint64(1)<<(p%64)
		if b.bits[w]&m == 0 {
			changed = true
			b.bits[w] |= m
		}
	}
	return changed
}
func (b *BloomFilter) Contains(data []byte) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, p := range b.locations(data) {
		if b.bits[p/64]&(uint64(1)<<(p%64)) == 0 {
			return false
		}
	}
	return true
}

type CountingBloomFilter struct {
	mu       sync.RWMutex
	counters []uint16
	K        uint32
	M        uint64
}

func NewCountingBloomFilter(expected uint64, falsePositiveRate float64) (*CountingBloomFilter, error) {
	b, e := NewBloomFilter(expected, falsePositiveRate)
	if e != nil {
		return nil, e
	}
	return &CountingBloomFilter{counters: make([]uint16, b.M), K: b.K, M: b.M}, nil
}
func (b *CountingBloomFilter) locations(data []byte) []uint64 {
	h1 := hash64(3, data)
	h2 := hash64(4, data) | 1
	out := make([]uint64, b.K)
	for i := uint32(0); i < b.K; i++ {
		out[i] = (h1 + uint64(i)*h2) % b.M
	}
	return out
}
func (b *CountingBloomFilter) Add(data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, p := range b.locations(data) {
		if b.counters[p] < 65535 {
			b.counters[p]++
		}
	}
}
func (b *CountingBloomFilter) Remove(data []byte) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	loc := b.locations(data)
	for _, p := range loc {
		if b.counters[p] == 0 {
			return false
		}
	}
	for _, p := range loc {
		b.counters[p]--
	}
	return true
}
func (b *CountingBloomFilter) Contains(data []byte) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, p := range b.locations(data) {
		if b.counters[p] == 0 {
			return false
		}
	}
	return true
}

type cuckooBucket [4]uint16
type CuckooFilter struct {
	mu       sync.RWMutex
	buckets  []cuckooBucket
	count    uint64
	maxKicks int
}

func NewCuckooFilter(capacity uint64) *CuckooFilter {
	n := uint64(2)
	for n < capacity/3 {
		n <<= 1
	}
	return &CuckooFilter{buckets: make([]cuckooBucket, n), maxKicks: 500}
}
func cuckooFingerprint(data []byte) uint16 {
	f := uint16(hash64(9, data))
	if f == 0 {
		f = 1
	}
	return f
}
func (c *CuckooFilter) indexes(data []byte, fp uint16) (uint64, uint64) {
	i1 := hash64(10, data) & uint64(len(c.buckets)-1)
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], fp)
	i2 := (i1 ^ hash64(11, b[:])) & uint64(len(c.buckets)-1)
	return i1, i2
}
func bucketContains(b cuckooBucket, fp uint16) bool {
	for _, x := range b {
		if x == fp {
			return true
		}
	}
	return false
}
func bucketInsert(b *cuckooBucket, fp uint16) bool {
	for i, x := range b {
		if x == 0 {
			b[i] = fp
			return true
		}
	}
	return false
}
func (c *CuckooFilter) Add(data []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	fp := cuckooFingerprint(data)
	i1, i2 := c.indexes(data, fp)
	if bucketContains(c.buckets[i1], fp) || bucketContains(c.buckets[i2], fp) {
		return true
	}
	if bucketInsert(&c.buckets[i1], fp) || bucketInsert(&c.buckets[i2], fp) {
		c.count++
		return true
	}
	idx := i1
	cur := fp
	for kick := 0; kick < c.maxKicks; kick++ {
		slot := int((hash64(uint64(kick+20), data) + uint64(cur)) % 4)
		cur, c.buckets[idx][slot] = c.buckets[idx][slot], cur
		var bb [2]byte
		binary.BigEndian.PutUint16(bb[:], cur)
		idx = (idx ^ hash64(11, bb[:])) & uint64(len(c.buckets)-1)
		if bucketInsert(&c.buckets[idx], cur) {
			c.count++
			return true
		}
	}
	return false
}
func (c *CuckooFilter) Contains(data []byte) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	fp := cuckooFingerprint(data)
	i1, i2 := c.indexes(data, fp)
	return bucketContains(c.buckets[i1], fp) || bucketContains(c.buckets[i2], fp)
}
func (c *CuckooFilter) Remove(data []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	fp := cuckooFingerprint(data)
	i1, i2 := c.indexes(data, fp)
	for _, idx := range []uint64{i1, i2} {
		for j, x := range c.buckets[idx] {
			if x == fp {
				c.buckets[idx][j] = 0
				c.count--
				return true
			}
		}
	}
	return false
}

type CountMinSketch struct {
	mu           sync.RWMutex
	Width, Depth uint32
	table        [][]uint64
}

func NewCountMinSketch(width, depth uint32) (*CountMinSketch, error) {
	if width == 0 || depth == 0 {
		return nil, fmt.Errorf("width/depth required")
	}
	t := make([][]uint64, depth)
	for i := range t {
		t[i] = make([]uint64, width)
	}
	return &CountMinSketch{Width: width, Depth: depth, table: t}, nil
}
func (c *CountMinSketch) Add(data []byte, n uint64) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	min := uint64(math.MaxUint64)
	for d := uint32(0); d < c.Depth; d++ {
		i := hash64(uint64(100+d), data) % uint64(c.Width)
		c.table[d][i] += n
		if c.table[d][i] < min {
			min = c.table[d][i]
		}
	}
	return min
}
func (c *CountMinSketch) Estimate(data []byte) uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	min := uint64(math.MaxUint64)
	for d := uint32(0); d < c.Depth; d++ {
		i := hash64(uint64(100+d), data) % uint64(c.Width)
		if c.table[d][i] < min {
			min = c.table[d][i]
		}
	}
	if min == math.MaxUint64 {
		return 0
	}
	return min
}

type TopKItem struct {
	Item  string `json:"item"`
	Count uint64 `json:"count"`
}
type TopK struct {
	mu            sync.RWMutex
	K             int
	maxCandidates int
	counts        map[string]uint64
	sketch        *CountMinSketch
}

func NewTopK(k int) *TopK {
	if k < 1 {
		k = 1
	}
	s, _ := NewCountMinSketch(2048, 5)
	cap := k * 4
	if cap < k+8 {
		cap = k + 8
	}
	return &TopK{K: k, maxCandidates: cap, counts: map[string]uint64{}, sketch: s}
}
func (t *TopK) Add(item string, n uint64) uint64 {
	if n == 0 {
		return t.Estimate(item)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	est := t.sketch.Add([]byte(item), n)
	if _, ok := t.counts[item]; ok {
		t.counts[item] = est
		return est
	}
	if len(t.counts) < t.maxCandidates {
		t.counts[item] = est
		return est
	}
	var minItem string
	minCount := uint64(math.MaxUint64)
	for k, v := range t.counts {
		if v < minCount || v == minCount && k > minItem {
			minItem, minCount = k, v
		}
	}
	if est > minCount {
		delete(t.counts, minItem)
		t.counts[item] = est
	}
	return est
}
func (t *TopK) Estimate(item string) uint64 { return t.sketch.Estimate([]byte(item)) }
func (t *TopK) Items() []TopKItem {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]TopKItem, 0, len(t.counts))
	for k, v := range t.counts {
		out = append(out, TopKItem{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Item < out[j].Item
		}
		return out[i].Count > out[j].Count
	})
	if len(out) > t.K {
		out = out[:t.K]
	}
	return out
}

type centroid struct{ Mean, Weight float64 }
type TDigest struct {
	mu          sync.Mutex
	Compression float64
	centroids   []centroid
	total       float64
}

func NewTDigest(compression float64) *TDigest {
	if compression < 20 {
		compression = 100
	}
	return &TDigest{Compression: compression}
}
func (t *TDigest) Add(x float64, weight float64) {
	if weight <= 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.centroids = append(t.centroids, centroid{x, weight})
	t.total += weight
	if len(t.centroids) > int(t.Compression*10) {
		t.compressLocked()
	}
}
func (t *TDigest) compressLocked() {
	if len(t.centroids) < 2 {
		return
	}
	sort.Slice(t.centroids, func(i, j int) bool { return t.centroids[i].Mean < t.centroids[j].Mean })
	out := make([]centroid, 0, int(t.Compression)+8)
	cum := 0.0
	cur := t.centroids[0]
	for _, c := range t.centroids[1:] {
		q := (cum + cur.Weight/2) / math.Max(t.total, 1)
		maxW := 4 * t.total * q * (1 - q) / t.Compression
		if maxW < 1 {
			maxW = 1
		}
		if cur.Weight+c.Weight <= maxW {
			w := cur.Weight + c.Weight
			cur.Mean = (cur.Mean*cur.Weight + c.Mean*c.Weight) / w
			cur.Weight = w
		} else {
			out = append(out, cur)
			cum += cur.Weight
			cur = c
		}
	}
	out = append(out, cur)
	t.centroids = out
}
func (t *TDigest) Quantile(q float64) float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.centroids) == 0 {
		return math.NaN()
	}
	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}
	t.compressLocked()
	sort.Slice(t.centroids, func(i, j int) bool { return t.centroids[i].Mean < t.centroids[j].Mean })
	if q == 0 {
		return t.centroids[0].Mean
	}
	if q == 1 {
		return t.centroids[len(t.centroids)-1].Mean
	}
	target := q * t.total
	cum := 0.0
	for i, c := range t.centroids {
		next := cum + c.Weight
		if target <= next {
			if i == 0 {
				return c.Mean
			}
			prev := t.centroids[i-1]
			span := (prev.Weight + c.Weight) / 2
			if span <= 0 {
				return c.Mean
			}
			frac := (target - (cum - prev.Weight/2)) / span
			if frac < 0 {
				frac = 0
			}
			if frac > 1 {
				frac = 1
			}
			return prev.Mean + (c.Mean-prev.Mean)*frac
		}
		cum = next
	}
	return t.centroids[len(t.centroids)-1].Mean
}
