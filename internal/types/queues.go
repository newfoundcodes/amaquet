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
	"container/heap"
	"context"
	"fmt"
	"github.com/newfoundcodes/amaquet/internal/core"
	"sync"
	"time"
)

type FIFOQueue struct {
	mu    sync.Mutex
	items []core.Value
}

func NewFIFOQueue() *FIFOQueue            { return &FIFOQueue{} }
func (q *FIFOQueue) Enqueue(v core.Value) { q.mu.Lock(); q.items = append(q.items, v); q.mu.Unlock() }
func (q *FIFOQueue) Dequeue() (core.Value, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return core.Value{}, false
	}
	v := q.items[0]
	q.items = q.items[1:]
	return v, true
}
func (q *FIFOQueue) Peek() (core.Value, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return core.Value{}, false
	}
	return q.items[0], true
}
func (q *FIFOQueue) Len() int { q.mu.Lock(); defer q.mu.Unlock(); return len(q.items) }

type LIFOStack struct {
	mu    sync.Mutex
	items []core.Value
}

func NewLIFOStack() *LIFOStack         { return &LIFOStack{} }
func (s *LIFOStack) Push(v core.Value) { s.mu.Lock(); s.items = append(s.items, v); s.mu.Unlock() }
func (s *LIFOStack) Pop() (core.Value, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.items)
	if n == 0 {
		return core.Value{}, false
	}
	v := s.items[n-1]
	s.items = s.items[:n-1]
	return v, true
}
func (s *LIFOStack) Peek() (core.Value, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(s.items)
	if n == 0 {
		return core.Value{}, false
	}
	return s.items[n-1], true
}
func (s *LIFOStack) Len() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.items) }

type PriorityItem struct {
	Value    core.Value `json:"value"`
	Priority float64    `json:"priority"`
	Sequence uint64     `json:"sequence"`
	index    int
}
type priorityHeap []*PriorityItem

func (h priorityHeap) Len() int { return len(h) }
func (h priorityHeap) Less(i, j int) bool {
	if h[i].Priority == h[j].Priority {
		return h[i].Sequence < h[j].Sequence
	}
	return h[i].Priority > h[j].Priority
}
func (h priorityHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i]; h[i].index = i; h[j].index = j }
func (h *priorityHeap) Push(x any)   { it := x.(*PriorityItem); it.index = len(*h); *h = append(*h, it) }
func (h *priorityHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return it
}

type PriorityQueue struct {
	mu  sync.Mutex
	h   priorityHeap
	seq uint64
}

func NewPriorityQueue() *PriorityQueue { q := &PriorityQueue{}; heap.Init(&q.h); return q }
func (q *PriorityQueue) Enqueue(v core.Value, priority float64) {
	q.mu.Lock()
	q.seq++
	heap.Push(&q.h, &PriorityItem{Value: v, Priority: priority, Sequence: q.seq})
	q.mu.Unlock()
}
func (q *PriorityQueue) Dequeue() (PriorityItem, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.h) == 0 {
		return PriorityItem{}, false
	}
	return *heap.Pop(&q.h).(*PriorityItem), true
}
func (q *PriorityQueue) Len() int { q.mu.Lock(); defer q.mu.Unlock(); return q.h.Len() }

type BlockingQueue struct {
	mu     sync.Mutex
	items  []core.Value
	notify chan struct{}
	closed bool
}

func NewBlockingQueue() *BlockingQueue { return &BlockingQueue{notify: make(chan struct{}, 1)} }
func (q *BlockingQueue) signal() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}
func (q *BlockingQueue) Enqueue(v core.Value) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return fmt.Errorf("queue closed")
	}
	q.items = append(q.items, v)
	q.signal()
	return nil
}
func (q *BlockingQueue) Dequeue(ctx context.Context) (core.Value, error) {
	for {
		q.mu.Lock()
		if len(q.items) > 0 {
			v := q.items[0]
			q.items = q.items[1:]
			q.mu.Unlock()
			return v, nil
		}
		if q.closed {
			q.mu.Unlock()
			return core.Value{}, fmt.Errorf("queue closed")
		}
		q.mu.Unlock()
		select {
		case <-ctx.Done():
			return core.Value{}, ctx.Err()
		case <-q.notify:
		}
	}
}
func (q *BlockingQueue) Close()   { q.mu.Lock(); q.closed = true; q.signal(); q.mu.Unlock() }
func (q *BlockingQueue) Len() int { q.mu.Lock(); defer q.mu.Unlock(); return len(q.items) }

type DelayedItem struct {
	ID          string     `json:"id"`
	Value       core.Value `json:"value"`
	AvailableAt time.Time  `json:"available_at"`
	Sequence    uint64     `json:"sequence"`
}
type DelayedQueue struct {
	mu    sync.Mutex
	items []DelayedItem
	seq   uint64
}

func NewDelayedQueue() *DelayedQueue { return &DelayedQueue{} }
func (q *DelayedQueue) Enqueue(id string, v core.Value, at time.Time) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.seq++
	q.items = append(q.items, DelayedItem{ID: id, Value: v, AvailableAt: at, Sequence: q.seq})
}
func (q *DelayedQueue) PopReady(now time.Time) (DelayedItem, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	best := -1
	for i, it := range q.items {
		if !it.AvailableAt.After(now) && (best < 0 || it.AvailableAt.Before(q.items[best].AvailableAt) || it.AvailableAt.Equal(q.items[best].AvailableAt) && it.Sequence < q.items[best].Sequence) {
			best = i
		}
	}
	if best < 0 {
		return DelayedItem{}, false
	}
	it := q.items[best]
	q.items = append(q.items[:best], q.items[best+1:]...)
	return it, true
}
func (q *DelayedQueue) Len() int { q.mu.Lock(); defer q.mu.Unlock(); return len(q.items) }

type ReliableMessage struct {
	ID          string     `json:"id"`
	Value       core.Value `json:"value"`
	Attempts    int        `json:"attempts"`
	EnqueuedAt  time.Time  `json:"enqueued_at"`
	LeasedUntil time.Time  `json:"leased_until,omitempty"`
}
type ReliableQueue struct {
	mu          sync.Mutex
	ready       []ReliableMessage
	pending     map[string]ReliableMessage
	dead        []ReliableMessage
	MaxAttempts int
}

func NewReliableQueue(maxAttempts int) *ReliableQueue {
	if maxAttempts < 1 {
		maxAttempts = 5
	}
	return &ReliableQueue{pending: map[string]ReliableMessage{}, MaxAttempts: maxAttempts}
}
func (q *ReliableQueue) Enqueue(id string, v core.Value) { q.EnqueueAt(id, v, time.Now().UTC()) }
func (q *ReliableQueue) EnqueueAt(id string, v core.Value, at time.Time) {
	q.mu.Lock()
	q.ready = append(q.ready, ReliableMessage{ID: id, Value: v, EnqueuedAt: at.UTC()})
	q.mu.Unlock()
}
func (q *ReliableQueue) Claim(lease time.Duration) (ReliableMessage, bool) {
	return q.ClaimAt(lease, time.Now())
}
func (q *ReliableQueue) ClaimAt(lease time.Duration, now time.Time) (ReliableMessage, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.requeueExpiredLocked(now)
	if len(q.ready) == 0 {
		return ReliableMessage{}, false
	}
	m := q.ready[0]
	q.ready = q.ready[1:]
	m.Attempts++
	m.LeasedUntil = now.Add(lease)
	q.pending[m.ID] = m
	return m, true
}
func (q *ReliableQueue) Ack(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, ok := q.pending[id]; !ok {
		return false
	}
	delete(q.pending, id)
	return true
}
func (q *ReliableQueue) Nack(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	m, ok := q.pending[id]
	if !ok {
		return false
	}
	delete(q.pending, id)
	m.LeasedUntil = time.Time{}
	if m.Attempts >= q.MaxAttempts {
		q.dead = append(q.dead, m)
	} else {
		q.ready = append(q.ready, m)
	}
	return true
}
func (q *ReliableQueue) requeueExpiredLocked(now time.Time) {
	for id, m := range q.pending {
		if !m.LeasedUntil.After(now) {
			delete(q.pending, id)
			m.LeasedUntil = time.Time{}
			if m.Attempts >= q.MaxAttempts {
				q.dead = append(q.dead, m)
			} else {
				q.ready = append(q.ready, m)
			}
		}
	}
}
func (q *ReliableQueue) Stats() (ready, pending, dead int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.requeueExpiredLocked(time.Now())
	return len(q.ready), len(q.pending), len(q.dead)
}
