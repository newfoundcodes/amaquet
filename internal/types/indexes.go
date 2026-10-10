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
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

type btreeNode struct {
	leaf     bool
	keys     []string
	values   []map[string]struct{}
	children []*btreeNode
}
type BTreeIndex struct {
	mu      sync.RWMutex
	Degree  int
	root    *btreeNode
	entries map[string]map[string]struct{}
}

func NewBTreeIndex(degree int) *BTreeIndex {
	if degree < 2 {
		degree = 16
	}
	return &BTreeIndex{Degree: degree, root: &btreeNode{leaf: true}, entries: map[string]map[string]struct{}{}}
}
func cloneStringSet(s map[string]struct{}) []string {
	out := make([]string, 0, len(s))
	for x := range s {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}
func (b *BTreeIndex) Search(key string) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	n := b.root
	for n != nil {
		i := sort.SearchStrings(n.keys, key)
		if i < len(n.keys) && n.keys[i] == key {
			return cloneStringSet(n.values[i])
		}
		if n.leaf {
			return nil
		}
		n = n.children[i]
	}
	return nil
}
func (b *BTreeIndex) Insert(key, id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	set := b.entries[key]
	if set == nil {
		set = map[string]struct{}{}
		b.entries[key] = set
	}
	if _, ok := set[id]; ok {
		return
	}
	set[id] = struct{}{}
	if len(b.root.keys) == 2*b.Degree-1 {
		r := &btreeNode{leaf: false, children: []*btreeNode{b.root}}
		b.splitChild(r, 0)
		b.root = r
	}
	b.insertNonFull(b.root, key, id)
}
func (b *BTreeIndex) insertNonFull(n *btreeNode, key, id string) {
	i := sort.SearchStrings(n.keys, key)
	if i < len(n.keys) && n.keys[i] == key {
		n.values[i][id] = struct{}{}
		return
	}
	if n.leaf {
		n.keys = append(n.keys, "")
		copy(n.keys[i+1:], n.keys[i:])
		n.keys[i] = key
		n.values = append(n.values, nil)
		copy(n.values[i+1:], n.values[i:])
		n.values[i] = map[string]struct{}{id: {}}
		return
	}
	if len(n.children[i].keys) == 2*b.Degree-1 {
		b.splitChild(n, i)
		if key > n.keys[i] {
			i++
		} else if key == n.keys[i] {
			n.values[i][id] = struct{}{}
			return
		}
	}
	b.insertNonFull(n.children[i], key, id)
}
func (b *BTreeIndex) splitChild(parent *btreeNode, i int) {
	t := b.Degree
	y := parent.children[i]
	z := &btreeNode{leaf: y.leaf}
	midKey := y.keys[t-1]
	midVal := y.values[t-1]
	z.keys = append(z.keys, y.keys[t:]...)
	z.values = append(z.values, y.values[t:]...)
	y.keys = y.keys[:t-1]
	y.values = y.values[:t-1]
	if !y.leaf {
		z.children = append(z.children, y.children[t:]...)
		y.children = y.children[:t]
	}
	parent.keys = append(parent.keys, "")
	copy(parent.keys[i+1:], parent.keys[i:])
	parent.keys[i] = midKey
	parent.values = append(parent.values, nil)
	copy(parent.values[i+1:], parent.values[i:])
	parent.values[i] = midVal
	parent.children = append(parent.children, nil)
	copy(parent.children[i+2:], parent.children[i+1:])
	parent.children[i+1] = z
}
func (b *BTreeIndex) Delete(key, id string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	set := b.entries[key]
	if set == nil {
		return false
	}
	if _, ok := set[id]; !ok {
		return false
	}
	delete(set, id)
	if len(set) == 0 {
		delete(b.entries, key)
	}
	b.rebuildLocked()
	return true
}
func (b *BTreeIndex) rebuildLocked() {
	b.root = &btreeNode{leaf: true}
	keys := make([]string, 0, len(b.entries))
	for k := range b.entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for id := range b.entries[k] {
			if len(b.root.keys) == 2*b.Degree-1 {
				r := &btreeNode{leaf: false, children: []*btreeNode{b.root}}
				b.splitChild(r, 0)
				b.root = r
			}
			b.insertNonFull(b.root, k, id)
		}
	}
}
func (b *BTreeIndex) Range(min, max string, limit int) map[string][]string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := map[string][]string{}
	count := 0
	var walk func(*btreeNode) bool
	walk = func(n *btreeNode) bool {
		if n == nil {
			return false
		}
		for i, k := range n.keys {
			if !n.leaf {
				if walk(n.children[i]) {
					return true
				}
			}
			if (min == "" || k >= min) && (max == "" || k <= max) {
				out[k] = cloneStringSet(n.values[i])
				count++
				if limit > 0 && count >= limit {
					return true
				}
			}
			if max != "" && k > max {
				return true
			}
		}
		if !n.leaf {
			return walk(n.children[len(n.keys)])
		}
		return false
	}
	walk(b.root)
	return out
}

type HashIndex struct {
	mu    sync.RWMutex
	items map[string]map[string]struct{}
}

func NewHashIndex() *HashIndex { return &HashIndex{items: map[string]map[string]struct{}{}} }
func (h *HashIndex) Add(value, id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.items[value] == nil {
		h.items[value] = map[string]struct{}{}
	}
	h.items[value][id] = struct{}{}
}
func (h *HashIndex) Remove(value, id string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.items[value]
	if _, ok := s[id]; !ok {
		return false
	}
	delete(s, id)
	if len(s) == 0 {
		delete(h.items, value)
	}
	return true
}
func (h *HashIndex) Lookup(value string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return cloneStringSet(h.items[value])
}

type radixNode struct {
	children map[byte]*radixNode
	terminal bool
	values   map[string]struct{}
}
type RadixTree struct {
	mu   sync.RWMutex
	root *radixNode
}

func NewRadixTree() *RadixTree { return &RadixTree{root: &radixNode{children: map[byte]*radixNode{}}} }
func (r *RadixTree) Insert(key, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := r.root
	for i := 0; i < len(key); i++ {
		if n.children[key[i]] == nil {
			n.children[key[i]] = &radixNode{children: map[byte]*radixNode{}}
		}
		n = n.children[key[i]]
	}
	n.terminal = true
	if n.values == nil {
		n.values = map[string]struct{}{}
	}
	n.values[id] = struct{}{}
}
func (r *RadixTree) Exact(key string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := r.root
	for i := 0; i < len(key); i++ {
		n = n.children[key[i]]
		if n == nil {
			return nil
		}
	}
	if !n.terminal {
		return nil
	}
	return cloneStringSet(n.values)
}
func (r *RadixTree) Prefix(prefix string, limit int) map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := r.root
	for i := 0; i < len(prefix); i++ {
		n = n.children[prefix[i]]
		if n == nil {
			return nil
		}
	}
	out := map[string][]string{}
	var walk func(*radixNode, string)
	walk = func(x *radixNode, k string) {
		if limit > 0 && len(out) >= limit {
			return
		}
		if x.terminal {
			out[k] = cloneStringSet(x.values)
		}
		ks := make([]int, 0, len(x.children))
		for c := range x.children {
			ks = append(ks, int(c))
		}
		sort.Ints(ks)
		for _, c := range ks {
			walk(x.children[byte(c)], k+string(byte(c)))
		}
	}
	walk(n, prefix)
	return out
}

type Trie = RadixTree

func NewTrie() *Trie { return NewRadixTree() }

type InvertedIndex struct {
	mu       sync.RWMutex
	terms    map[string]map[string]uint32
	docTerms map[string]map[string]uint32
}

func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{terms: map[string]map[string]uint32{}, docTerms: map[string]map[string]uint32{}}
}
func tokenize(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}
func (i *InvertedIndex) Index(id, text string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.removeLocked(id)
	freq := map[string]uint32{}
	for _, t := range tokenize(text) {
		freq[t]++
	}
	i.docTerms[id] = freq
	for t, c := range freq {
		if i.terms[t] == nil {
			i.terms[t] = map[string]uint32{}
		}
		i.terms[t][id] = c
	}
}
func (i *InvertedIndex) removeLocked(id string) {
	for t := range i.docTerms[id] {
		delete(i.terms[t], id)
		if len(i.terms[t]) == 0 {
			delete(i.terms, t)
		}
	}
	delete(i.docTerms, id)
}
func (i *InvertedIndex) Remove(id string) { i.mu.Lock(); i.removeLocked(id); i.mu.Unlock() }

type TextHit struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

func (i *InvertedIndex) Search(query string, limit int) []TextHit {
	i.mu.RLock()
	defer i.mu.RUnlock()
	scores := map[string]float64{}
	terms := tokenize(query)
	nDocs := float64(len(i.docTerms))
	for _, t := range terms {
		post := i.terms[t]
		if len(post) == 0 {
			continue
		}
		idf := 1 + mathLog((nDocs+1)/(float64(len(post))+1))
		for id, tf := range post {
			scores[id] += float64(tf) * idf
		}
	}
	out := make([]TextHit, 0, len(scores))
	for id, s := range scores {
		out = append(out, TextHit{id, s})
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Score == out[b].Score {
			return out[a].ID < out[b].ID
		}
		return out[a].Score > out[b].Score
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
func mathLog(v float64) float64 {
	if v <= 0 {
		return 0
	}

	x := (v - 1) / (v + 1)
	x2 := x * x
	term := x
	sum := 0.0
	for k := 1; k <= 39; k += 2 {
		sum += term / float64(k)
		term *= x2
	}
	return 2 * sum
}

type SecondaryIndex struct {
	mu     sync.RWMutex
	fields map[string]*HashIndex
}

func NewSecondaryIndex() *SecondaryIndex { return &SecondaryIndex{fields: map[string]*HashIndex{}} }
func (s *SecondaryIndex) Add(field, value, id string) {
	s.mu.Lock()
	idx := s.fields[field]
	if idx == nil {
		idx = NewHashIndex()
		s.fields[field] = idx
	}
	s.mu.Unlock()
	idx.Add(value, id)
}
func (s *SecondaryIndex) Lookup(field, value string) []string {
	s.mu.RLock()
	idx := s.fields[field]
	s.mu.RUnlock()
	if idx == nil {
		return nil
	}
	return idx.Lookup(value)
}

type TagField struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}
type NumericField struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}
type TextField struct {
	Name   string  `json:"name"`
	Value  string  `json:"value"`
	Weight float64 `json:"weight"`
}
type FullTextDocument struct {
	mu            sync.RWMutex
	ID            string         `json:"id"`
	TextFields    []TextField    `json:"text_fields"`
	TagFields     []TagField     `json:"tag_fields"`
	NumericFields []NumericField `json:"numeric_fields"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func (d *FullTextDocument) AddText(f TextField) {
	d.mu.Lock()
	d.TextFields = append(d.TextFields, f)
	d.mu.Unlock()
}
func (d *FullTextDocument) AddTag(f TagField) {
	d.mu.Lock()
	d.TagFields = append(d.TagFields, f)
	d.mu.Unlock()
}
func (d *FullTextDocument) AddNumeric(f NumericField) {
	d.mu.Lock()
	d.NumericFields = append(d.NumericFields, f)
	d.mu.Unlock()
}
func (d *FullTextDocument) Snapshot() any {
	d.mu.RLock()
	defer d.mu.RUnlock()
	meta := map[string]any{}
	for k, v := range d.Metadata {
		meta[k] = v
	}
	return map[string]any{"id": d.ID, "text_fields": append([]TextField(nil), d.TextFields...), "tag_fields": append([]TagField(nil), d.TagFields...), "numeric_fields": append([]NumericField(nil), d.NumericFields...), "metadata": meta}
}
func (d *FullTextDocument) CombinedText() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var b strings.Builder
	for _, f := range d.TextFields {
		w := f.Weight
		if w <= 0 {
			w = 1
		}
		reps := int(w + 0.5)
		if reps < 1 {
			reps = 1
		}
		for j := 0; j < reps; j++ {
			b.WriteString(f.Value)
			b.WriteByte(' ')
		}
	}
	return b.String()
}

type GeospatialSecondaryIndex = GeoSpatialIndex

type AdjacencySet struct {
	mu    sync.RWMutex
	edges map[string]map[string]float64
}

func NewAdjacencySet() *AdjacencySet { return &AdjacencySet{edges: map[string]map[string]float64{}} }
func (a *AdjacencySet) Add(from, to string, weight float64) {
	a.mu.Lock()
	if a.edges[from] == nil {
		a.edges[from] = map[string]float64{}
	}
	a.edges[from][to] = weight
	a.mu.Unlock()
}
func (a *AdjacencySet) Remove(from, to string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.edges[from][to]; !ok {
		return false
	}
	delete(a.edges[from], to)
	if len(a.edges[from]) == 0 {
		delete(a.edges, from)
	}
	return true
}
func (a *AdjacencySet) Neighbors(from string) map[string]float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := map[string]float64{}
	for k, v := range a.edges[from] {
		out[k] = v
	}
	return out
}

func NumericKey(v float64) string { return fmt.Sprintf("%024.12f", v) }
func IntKey(v int64) string       { return strconv.FormatInt(v, 10) }
