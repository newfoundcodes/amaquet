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
	"hash/fnv"
	"math"
	"sort"
	"sync"
)

type DenseVector []float32

func (v DenseVector) Dim() int           { return len(v) }
func (v DenseVector) Clone() DenseVector { return append(DenseVector(nil), v...) }

type SparseVector struct {
	Dim    int             `json:"dim"`
	Values map[int]float32 `json:"values"`
}

func NewSparseVector(dim int, values map[int]float32) (SparseVector, error) {
	if dim <= 0 {
		return SparseVector{}, fmt.Errorf("dimension must be positive")
	}
	cp := map[int]float32{}
	for i, v := range values {
		if i < 0 || i >= dim {
			return SparseVector{}, fmt.Errorf("index %d out of range", i)
		}
		if v != 0 {
			cp[i] = v
		}
	}
	return SparseVector{dim, cp}, nil
}

type VectorMetric string

const (
	MetricCosine    VectorMetric = "cosine"
	MetricDot       VectorMetric = "dot"
	MetricEuclidean VectorMetric = "euclidean"
	MetricManhattan VectorMetric = "manhattan"
)

func VectorDistance(a, b DenseVector, metric VectorMetric) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("dimension mismatch")
	}
	switch metric {
	case MetricCosine:
		var dot, aa, bb float64
		for i := range a {
			x, y := float64(a[i]), float64(b[i])
			dot += x * y
			aa += x * x
			bb += y * y
		}
		if aa == 0 || bb == 0 {
			return 1, nil
		}
		return 1 - dot/(math.Sqrt(aa)*math.Sqrt(bb)), nil
	case MetricDot:
		var dot float64
		for i := range a {
			dot += float64(a[i]) * float64(b[i])
		}
		return -dot, nil
	case MetricEuclidean:
		var s float64
		for i := range a {
			d := float64(a[i] - b[i])
			s += d * d
		}
		return math.Sqrt(s), nil
	case MetricManhattan:
		var s float64
		for i := range a {
			s += math.Abs(float64(a[i] - b[i]))
		}
		return s, nil
	default:
		return 0, fmt.Errorf("unsupported metric %q", metric)
	}
}

type QuantizedVector struct {
	Scale     float32 `json:"scale"`
	ZeroPoint int8    `json:"zero_point"`
	Data      []int8  `json:"data"`
}

func QuantizeVector(v DenseVector) QuantizedVector {
	if len(v) == 0 {
		return QuantizedVector{Scale: 1}
	}
	maxAbs := float32(0)
	for _, x := range v {
		if x < 0 {
			x = -x
		}
		if x > maxAbs {
			maxAbs = x
		}
	}
	scale := maxAbs / 127
	if scale == 0 {
		scale = 1
	}
	q := make([]int8, len(v))
	for i, x := range v {
		n := int(math.Round(float64(x / scale)))
		if n > 127 {
			n = 127
		}
		if n < -127 {
			n = -127
		}
		q[i] = int8(n)
	}
	return QuantizedVector{Scale: scale, Data: q}
}
func (q QuantizedVector) Dequantize() DenseVector {
	v := make(DenseVector, len(q.Data))
	for i, x := range q.Data {
		v[i] = (float32(x) - float32(q.ZeroPoint)) * q.Scale
	}
	return v
}

type VectorMetadata map[string]any
type VectorItem struct {
	ID       string         `json:"id"`
	Vector   DenseVector    `json:"vector"`
	Metadata VectorMetadata `json:"metadata,omitempty"`
}
type VectorResult struct {
	ID       string         `json:"id"`
	Distance float64        `json:"distance"`
	Metadata VectorMetadata `json:"metadata,omitempty"`
}
type MetadataFilter func(VectorMetadata) bool

type VectorSet struct {
	mu     sync.RWMutex
	Dim    int
	Metric VectorMetric
	items  map[string]VectorItem
}

func NewVectorSet(dim int, metric VectorMetric) (*VectorSet, error) {
	if dim <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}
	return &VectorSet{Dim: dim, Metric: metric, items: map[string]VectorItem{}}, nil
}
func cloneMetadata(m VectorMetadata) VectorMetadata {
	cp := VectorMetadata{}
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
func (s *VectorSet) Add(id string, v DenseVector, metadata VectorMetadata) error {
	if len(v) != s.Dim {
		return fmt.Errorf("expected dimension %d", s.Dim)
	}
	s.mu.Lock()
	s.items[id] = VectorItem{ID: id, Vector: v.Clone(), Metadata: cloneMetadata(metadata)}
	s.mu.Unlock()
	return nil
}
func (s *VectorSet) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.items[id]
	delete(s.items, id)
	return ok
}
func (s *VectorSet) Get(id string) (VectorItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	it, ok := s.items[id]
	if ok {
		it.Vector = it.Vector.Clone()
		it.Metadata = cloneMetadata(it.Metadata)
	}
	return it, ok
}
func (s *VectorSet) Search(q DenseVector, k int, filter MetadataFilter) ([]VectorResult, error) {
	if len(q) != s.Dim {
		return nil, fmt.Errorf("expected dimension %d", s.Dim)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]VectorResult, 0, len(s.items))
	for _, it := range s.items {
		if filter != nil && !filter(it.Metadata) {
			continue
		}
		d, e := VectorDistance(q, it.Vector, s.Metric)
		if e != nil {
			return nil, e
		}
		out = append(out, VectorResult{it.ID, d, cloneMetadata(it.Metadata)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Distance < out[j].Distance })
	if k > 0 && len(out) > k {
		out = out[:k]
	}
	return out, nil
}
func (s *VectorSet) Len() int { s.mu.RLock(); defer s.mu.RUnlock(); return len(s.items) }

type FlatVectorIndex struct{ Set *VectorSet }

func NewFlatVectorIndex(dim int, metric VectorMetric) (*FlatVectorIndex, error) {
	s, e := NewVectorSet(dim, metric)
	if e != nil {
		return nil, e
	}
	return &FlatVectorIndex{Set: s}, nil
}

type HNSWIndex struct {
	mu             sync.RWMutex
	Dim            int
	Metric         VectorMetric
	M              int
	efSearch       int
	efConstruction int
	vectors        map[string]DenseVector
	meta           map[string]VectorMetadata
	levels         map[string]int
	neighbors      map[int]map[string]map[string]float64
	entry          string
	maxLevel       int
}

type hnswCandidate struct {
	id string
	d  float64
}

func NewHNSWIndex(dim int, metric VectorMetric, m, efSearch int) (*HNSWIndex, error) {
	if dim <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}
	if m < 2 {
		m = 16
	}
	if efSearch < m {
		efSearch = 64
	}
	return &HNSWIndex{Dim: dim, Metric: metric, M: m, efSearch: efSearch, efConstruction: maxInt(efSearch*2, m*4), vectors: map[string]DenseVector{}, meta: map[string]VectorMetadata{}, levels: map[string]int{}, neighbors: map[int]map[string]map[string]float64{}, maxLevel: -1}, nil
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func hnswLevel(id string, m int) int {
	h := fnv.New64a()
	_, _ = h.Write([]byte(id))
	x := h.Sum64()
	level := 0
	threshold := uint64(math.MaxUint64 / uint64(maxInt(m, 2)))
	for level < 16 && x < threshold {
		level++
		x = (x*6364136223846793005 + 1442695040888963407)
	}
	return level
}
func (h *HNSWIndex) layerLocked(level int) map[string]map[string]float64 {
	l := h.neighbors[level]
	if l == nil {
		l = map[string]map[string]float64{}
		h.neighbors[level] = l
	}
	return l
}
func (h *HNSWIndex) distanceLocked(q DenseVector, id string) float64 {
	d, _ := VectorDistance(q, h.vectors[id], h.Metric)
	return d
}
func (h *HNSWIndex) greedyLocked(q DenseVector, entry string, level int) string {
	cur := entry
	if cur == "" {
		return cur
	}
	best := h.distanceLocked(q, cur)
	for {
		improved := false
		for nid := range h.neighbors[level][cur] {
			d := h.distanceLocked(q, nid)
			if d < best {
				best = d
				cur = nid
				improved = true
			}
		}
		if !improved {
			return cur
		}
	}
}
func (h *HNSWIndex) searchLayerLocked(q DenseVector, entries []string, level, ef int) []hnswCandidate {
	if ef < 1 {
		ef = 1
	}
	visited := map[string]bool{}
	front := make([]hnswCandidate, 0, ef)
	best := make([]hnswCandidate, 0, ef)
	push := func(id string) {
		if id == "" || visited[id] {
			return
		}
		visited[id] = true
		c := hnswCandidate{id, h.distanceLocked(q, id)}
		front = append(front, c)
		best = append(best, c)
	}
	for _, id := range entries {
		push(id)
	}
	for len(front) > 0 && len(visited) <= ef*8 {
		sort.Slice(front, func(i, j int) bool { return front[i].d < front[j].d })
		c := front[0]
		front = front[1:]
		for nid := range h.neighbors[level][c.id] {
			if !visited[nid] {
				push(nid)
			}
		}
		sort.Slice(best, func(i, j int) bool { return best[i].d < best[j].d })
		if len(best) > ef {
			best = best[:ef]
		}
	}
	sort.Slice(best, func(i, j int) bool { return best[i].d < best[j].d })
	return best
}
func (h *HNSWIndex) connectLevelLocked(level int, id string, cs []hnswCandidate) {
	l := h.layerLocked(level)
	if l[id] == nil {
		l[id] = map[string]float64{}
	}
	limit := h.M
	if level == 0 {
		limit = h.M * 2
	}
	if len(cs) > limit {
		cs = cs[:limit]
	}
	for _, c := range cs {
		if c.id == id {
			continue
		}
		l[id][c.id] = c.d
		if l[c.id] == nil {
			l[c.id] = map[string]float64{}
		}
		l[c.id][id] = c.d
		h.pruneLevelLocked(level, c.id, limit)
	}
	h.pruneLevelLocked(level, id, limit)
}
func (h *HNSWIndex) pruneLevelLocked(level int, id string, limit int) {
	n := h.neighbors[level][id]
	if len(n) <= limit {
		return
	}
	xs := make([]hnswCandidate, 0, len(n))
	for k, d := range n {
		xs = append(xs, hnswCandidate{k, d})
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i].d < xs[j].d })
	keep := map[string]float64{}
	for i := 0; i < limit; i++ {
		keep[xs[i].id] = xs[i].d
	}
	for k := range n {
		if _, ok := keep[k]; !ok {
			delete(h.neighbors[level][k], id)
		}
	}
	h.neighbors[level][id] = keep
}
func (h *HNSWIndex) Add(id string, v DenseVector, metadata VectorMetadata) error {
	if len(v) != h.Dim {
		return fmt.Errorf("expected dimension %d", h.Dim)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.vectors[id]; ok {
		h.removeLocked(id)
	}
	level := hnswLevel(id, h.M)
	h.vectors[id] = v.Clone()
	h.meta[id] = cloneMetadata(metadata)
	h.levels[id] = level
	for l := 0; l <= level; l++ {
		layer := h.layerLocked(l)
		layer[id] = map[string]float64{}
	}
	if h.entry == "" {
		h.entry = id
		h.maxLevel = level
		return nil
	}
	ep := h.entry
	for l := h.maxLevel; l > level; l-- {
		ep = h.greedyLocked(v, ep, l)
	}
	upper := level
	if h.maxLevel < upper {
		upper = h.maxLevel
	}
	for l := upper; l >= 0; l-- {
		cs := h.searchLayerLocked(v, []string{ep}, l, h.efConstruction)
		h.connectLevelLocked(l, id, cs)
		if len(cs) > 0 {
			ep = cs[0].id
		}
	}
	if level > h.maxLevel {
		h.entry = id
		h.maxLevel = level
	}
	return nil
}
func (h *HNSWIndex) removeLocked(id string) {
	lvl, ok := h.levels[id]
	if !ok {
		return
	}
	for l := 0; l <= lvl; l++ {
		for n := range h.neighbors[l][id] {
			delete(h.neighbors[l][n], id)
		}
		delete(h.neighbors[l], id)
	}
	delete(h.vectors, id)
	delete(h.meta, id)
	delete(h.levels, id)
	if h.entry == id {
		h.entry = ""
		h.maxLevel = -1
		for k, l := range h.levels {
			if l > h.maxLevel {
				h.entry = k
				h.maxLevel = l
			}
		}
	}
}
func (h *HNSWIndex) Remove(id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.vectors[id]; !ok {
		return false
	}
	h.removeLocked(id)
	return true
}
func (h *HNSWIndex) Search(q DenseVector, k int, filter MetadataFilter) ([]VectorResult, error) {
	if len(q) != h.Dim {
		return nil, fmt.Errorf("expected dimension %d", h.Dim)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.entry == "" {
		return nil, nil
	}
	ep := h.entry
	for l := h.maxLevel; l > 0; l-- {
		ep = h.greedyLocked(q, ep, l)
	}
	ef := h.efSearch
	if k > ef {
		ef = k
	}
	hits := h.searchLayerLocked(q, []string{ep}, 0, ef)
	out := make([]VectorResult, 0, k)
	for _, x := range hits {
		m := h.meta[x.id]
		if filter != nil && !filter(m) {
			continue
		}
		out = append(out, VectorResult{x.id, x.d, cloneMetadata(m)})
		if k > 0 && len(out) >= k {
			break
		}
	}
	return out, nil
}
