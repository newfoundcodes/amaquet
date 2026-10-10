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
	"github.com/newfoundcodes/amaquet/internal/core"
	"sort"
	"sync"
)

type List struct {
	mu    sync.RWMutex
	items []core.Value
}

func NewList(values ...core.Value) *List { return &List{items: cloneValues(values)} }
func (l *List) Len() int                 { l.mu.RLock(); defer l.mu.RUnlock(); return len(l.items) }
func (l *List) PushFront(v core.Value) {
	l.mu.Lock()
	l.items = append([]core.Value{v}, l.items...)
	l.mu.Unlock()
}
func (l *List) PushBack(v core.Value) { l.mu.Lock(); l.items = append(l.items, v); l.mu.Unlock() }
func (l *List) PopFront() (core.Value, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.items) == 0 {
		return core.Value{}, false
	}
	v := l.items[0]
	l.items = l.items[1:]
	return v, true
}
func (l *List) PopBack() (core.Value, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(l.items)
	if n == 0 {
		return core.Value{}, false
	}
	v := l.items[n-1]
	l.items = l.items[:n-1]
	return v, true
}
func (l *List) Range(start, stop int) []core.Value {
	l.mu.RLock()
	defer l.mu.RUnlock()
	n := len(l.items)
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= n {
		stop = n - 1
	}
	if start > stop || start >= n {
		return nil
	}
	return cloneValues(l.items[start : stop+1])
}

type Array struct {
	mu          sync.RWMutex
	ElementType core.DataType
	Typed       bool
	items       []core.Value
}

func NewArray(elementType core.DataType, typed bool, values ...core.Value) (*Array, error) {
	a := &Array{ElementType: elementType, Typed: typed}
	for _, v := range values {
		if err := a.Append(v); err != nil {
			return nil, err
		}
	}
	return a, nil
}
func (a *Array) check(v core.Value) error {
	if a.Typed && v.Type != a.ElementType {
		return fmt.Errorf("array expects %s, got %s", a.ElementType, v.Type)
	}
	return nil
}
func (a *Array) Append(v core.Value) error {
	if err := a.check(v); err != nil {
		return err
	}
	a.mu.Lock()
	a.items = append(a.items, v)
	a.mu.Unlock()
	return nil
}
func (a *Array) Set(i int, v core.Value) error {
	if err := a.check(v); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if i < 0 {
		i = len(a.items) + i
	}
	if i < 0 || i >= len(a.items) {
		return fmt.Errorf("index out of range")
	}
	a.items[i] = v
	return nil
}
func (a *Array) Get(i int) (core.Value, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if i < 0 {
		i = len(a.items) + i
	}
	if i < 0 || i >= len(a.items) {
		return core.Value{}, false
	}
	return a.items[i], true
}
func (a *Array) Len() int { a.mu.RLock(); defer a.mu.RUnlock(); return len(a.items) }
func (a *Array) Values() []core.Value {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return cloneValues(a.items)
}

type Deque struct {
	mu         sync.RWMutex
	buf        []core.Value
	head, size int
}

func NewDeque(capacity int) *Deque {
	if capacity < 8 {
		capacity = 8
	}
	return &Deque{buf: make([]core.Value, capacity)}
}
func (d *Deque) grow() {
	n := make([]core.Value, len(d.buf)*2)
	for i := 0; i < d.size; i++ {
		n[i] = d.buf[(d.head+i)%len(d.buf)]
	}
	d.buf = n
	d.head = 0
}
func (d *Deque) PushBack(v core.Value) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.size == len(d.buf) {
		d.grow()
	}
	d.buf[(d.head+d.size)%len(d.buf)] = v
	d.size++
}
func (d *Deque) PushFront(v core.Value) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.size == len(d.buf) {
		d.grow()
	}
	d.head = (d.head - 1 + len(d.buf)) % len(d.buf)
	d.buf[d.head] = v
	d.size++
}
func (d *Deque) PopFront() (core.Value, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.size == 0 {
		return core.Value{}, false
	}
	v := d.buf[d.head]
	d.head = (d.head + 1) % len(d.buf)
	d.size--
	return v, true
}
func (d *Deque) PopBack() (core.Value, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.size == 0 {
		return core.Value{}, false
	}
	i := (d.head + d.size - 1) % len(d.buf)
	v := d.buf[i]
	d.size--
	return v, true
}
func (d *Deque) Len() int { d.mu.RLock(); defer d.mu.RUnlock(); return d.size }

type RingBuffer struct {
	mu         sync.RWMutex
	buf        []core.Value
	head, size int
}

func NewRingBuffer(capacity int) *RingBuffer {
	if capacity < 1 {
		capacity = 1
	}
	return &RingBuffer{buf: make([]core.Value, capacity)}
}
func (r *RingBuffer) Push(v core.Value) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.size < len(r.buf) {
		r.buf[(r.head+r.size)%len(r.buf)] = v
		r.size++
		return
	}
	r.buf[r.head] = v
	r.head = (r.head + 1) % len(r.buf)
}
func (r *RingBuffer) Values() []core.Value {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]core.Value, r.size)
	for i := 0; i < r.size; i++ {
		out[i] = r.buf[(r.head+i)%len(r.buf)]
	}
	return out
}
func (r *RingBuffer) Len() int { r.mu.RLock(); defer r.mu.RUnlock(); return r.size }

type Tuple struct{ Items []core.Value }

func NewTuple(values ...core.Value) Tuple { return Tuple{Items: cloneValues(values)} }
func (t Tuple) Len() int                  { return len(t.Items) }
func (t Tuple) Get(i int) (core.Value, bool) {
	if i < 0 {
		i = len(t.Items) + i
	}
	if i < 0 || i >= len(t.Items) {
		return core.Value{}, false
	}
	return t.Items[i], true
}

type Set struct {
	mu    sync.RWMutex
	items map[string]core.Value
}

func NewSet() *Set { return &Set{items: map[string]core.Value{}} }
func (s *Set) Add(v core.Value) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := valueKey(v)
	_, ok := s.items[k]
	s.items[k] = v
	return !ok
}
func (s *Set) Remove(v core.Value) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := valueKey(v)
	_, ok := s.items[k]
	delete(s.items, k)
	return ok
}
func (s *Set) Contains(v core.Value) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.items[valueKey(v)]
	return ok
}
func (s *Set) Values() []core.Value {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]core.Value, 0, len(s.items))
	for _, v := range s.items {
		out = append(out, v)
	}
	return out
}
func (s *Set) Len() int { s.mu.RLock(); defer s.mu.RUnlock(); return len(s.items) }

type SortedSetEntry struct {
	Member core.Value `json:"member"`
	Score  float64    `json:"score"`
}
type SortedSet struct {
	mu    sync.RWMutex
	items map[string]SortedSetEntry
}

func NewSortedSet() *SortedSet { return &SortedSet{items: map[string]SortedSetEntry{}} }
func (z *SortedSet) Add(member core.Value, score float64) bool {
	z.mu.Lock()
	defer z.mu.Unlock()
	k := valueKey(member)
	_, ok := z.items[k]
	z.items[k] = SortedSetEntry{member, score}
	return !ok
}
func (z *SortedSet) Remove(member core.Value) bool {
	z.mu.Lock()
	defer z.mu.Unlock()
	k := valueKey(member)
	_, ok := z.items[k]
	delete(z.items, k)
	return ok
}
func (z *SortedSet) Score(member core.Value) (float64, bool) {
	z.mu.RLock()
	defer z.mu.RUnlock()
	v, ok := z.items[valueKey(member)]
	return v.Score, ok
}
func (z *SortedSet) RangeByScore(min, max float64) []SortedSetEntry {
	z.mu.RLock()
	defer z.mu.RUnlock()
	out := make([]SortedSetEntry, 0)
	for _, e := range z.items {
		if e.Score >= min && e.Score <= max {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return valueKey(out[i].Member) < valueKey(out[j].Member)
		}
		return out[i].Score < out[j].Score
	})
	return out
}
func (z *SortedSet) Range(start, stop int, reverse bool) []SortedSetEntry {
	all := z.RangeByScore(-1.7976931348623157e308, 1.7976931348623157e308)
	if reverse {
		for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
			all[i], all[j] = all[j], all[i]
		}
	}
	n := len(all)
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= n {
		stop = n - 1
	}
	if start > stop || start >= n {
		return nil
	}
	return all[start : stop+1]
}

type OrderedSet struct {
	mu    sync.RWMutex
	order []string
	items map[string]core.Value
}

func NewOrderedSet() *OrderedSet { return &OrderedSet{items: map[string]core.Value{}} }
func (s *OrderedSet) Add(v core.Value) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := valueKey(v)
	if _, ok := s.items[k]; ok {
		return false
	}
	s.items[k] = v
	s.order = append(s.order, k)
	return true
}
func (s *OrderedSet) Remove(v core.Value) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := valueKey(v)
	if _, ok := s.items[k]; !ok {
		return false
	}
	delete(s.items, k)
	for i, x := range s.order {
		if x == k {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true
}
func (s *OrderedSet) Values() []core.Value {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]core.Value, 0, len(s.order))
	for _, k := range s.order {
		out = append(out, s.items[k])
	}
	return out
}

type MultiSet struct {
	mu     sync.RWMutex
	counts map[string]uint64
	values map[string]core.Value
}

func NewMultiSet() *MultiSet {
	return &MultiSet{counts: map[string]uint64{}, values: map[string]core.Value{}}
}
func (m *MultiSet) Add(v core.Value, n uint64) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := valueKey(v)
	m.values[k] = v
	m.counts[k] += n
	return m.counts[k]
}
func (m *MultiSet) Remove(v core.Value, n uint64) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := valueKey(v)
	c := m.counts[k]
	if n >= c {
		delete(m.counts, k)
		delete(m.values, k)
		return 0
	}
	m.counts[k] = c - n
	return c - n
}
func (m *MultiSet) Count(v core.Value) uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counts[valueKey(v)]
}

type HashMap struct {
	mu    sync.RWMutex
	items map[string]core.Value
}

func NewHashMap() *HashMap                    { return &HashMap{items: map[string]core.Value{}} }
func (h *HashMap) Set(k string, v core.Value) { h.mu.Lock(); h.items[k] = v; h.mu.Unlock() }
func (h *HashMap) Get(k string) (core.Value, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	v, ok := h.items[k]
	return v, ok
}
func (h *HashMap) Delete(k string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.items[k]
	delete(h.items, k)
	return ok
}
func (h *HashMap) Keys() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]string, 0, len(h.items))
	for k := range h.items {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func (h *HashMap) Len() int { h.mu.RLock(); defer h.mu.RUnlock(); return len(h.items) }

type OrderedMap struct {
	mu    sync.RWMutex
	order []string
	items map[string]core.Value
}

func NewOrderedMap() *OrderedMap { return &OrderedMap{items: map[string]core.Value{}} }
func (m *OrderedMap) Set(k string, v core.Value) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[k]; !ok {
		m.order = append(m.order, k)
	}
	m.items[k] = v
}
func (m *OrderedMap) Get(k string) (core.Value, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[k]
	return v, ok
}
func (m *OrderedMap) Delete(k string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[k]; !ok {
		return false
	}
	delete(m.items, k)
	for i, x := range m.order {
		if x == k {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	return true
}
func (m *OrderedMap) Entries() []struct {
	Key   string
	Value core.Value
} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]struct {
		Key   string
		Value core.Value
	}, 0, len(m.order))
	for _, k := range m.order {
		out = append(out, struct {
			Key   string
			Value core.Value
		}{k, m.items[k]})
	}
	return out
}

type MultiMap struct {
	mu    sync.RWMutex
	items map[string][]core.Value
}

func NewMultiMap() *MultiMap { return &MultiMap{items: map[string][]core.Value{}} }
func (m *MultiMap) Add(k string, v core.Value) {
	m.mu.Lock()
	m.items[k] = append(m.items[k], v)
	m.mu.Unlock()
}
func (m *MultiMap) Get(k string) []core.Value {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneValues(m.items[k])
}
func (m *MultiMap) RemoveValue(k string, v core.Value) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	needle := valueKey(v)
	xs := m.items[k]
	for i, x := range xs {
		if valueKey(x) == needle {
			m.items[k] = append(xs[:i], xs[i+1:]...)
			if len(m.items[k]) == 0 {
				delete(m.items, k)
			}
			return true
		}
	}
	return false
}
