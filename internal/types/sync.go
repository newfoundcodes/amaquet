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
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type AtomicCounter struct{ v atomic.Int64 }

func NewAtomicCounter(initial int64) *AtomicCounter {
	c := &AtomicCounter{}
	c.v.Store(initial)
	return c
}
func (c *AtomicCounter) Add(delta int64) int64              { return c.v.Add(delta) }
func (c *AtomicCounter) Load() int64                        { return c.v.Load() }
func (c *AtomicCounter) CompareAndSwap(old, new int64) bool { return c.v.CompareAndSwap(old, new) }

type Lease struct {
	mu        sync.Mutex
	Owner     string
	Token     string
	ExpiresAt time.Time
}

func NewLease() *Lease { return &Lease{} }
func (l *Lease) Acquire(owner, token string, ttl time.Duration, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if ttl <= 0 {
		return false
	}
	if l.Token != "" && now.Before(l.ExpiresAt) {
		return false
	}
	l.Owner = owner
	l.Token = token
	l.ExpiresAt = now.Add(ttl)
	return true
}
func (l *Lease) Renew(token string, ttl time.Duration, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if token == "" || token != l.Token || !now.Before(l.ExpiresAt) || ttl <= 0 {
		return false
	}
	l.ExpiresAt = now.Add(ttl)
	return true
}
func (l *Lease) Release(token string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if token == "" || token != l.Token {
		return false
	}
	l.Owner = ""
	l.Token = ""
	l.ExpiresAt = time.Time{}
	return true
}
func (l *Lease) Held(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.Token != "" && now.Before(l.ExpiresAt)
}

type Lock struct{ lease *Lease }

func NewLock() *Lock { return &Lock{lease: NewLease()} }
func (l *Lock) TryLock(owner, token string, ttl time.Duration) bool {
	return l.lease.Acquire(owner, token, ttl, time.Now())
}
func (l *Lock) Unlock(token string) bool { return l.lease.Release(token) }
func (l *Lock) Renew(token string, ttl time.Duration) bool {
	return l.lease.Renew(token, ttl, time.Now())
}

type Barrier struct {
	mu         sync.Mutex
	target     int
	arrived    int
	generation uint64
	notify     chan struct{}
}

func NewBarrier(target int) (*Barrier, error) {
	if target < 1 {
		return nil, fmt.Errorf("target must be positive")
	}
	return &Barrier{target: target, notify: make(chan struct{})}, nil
}
func (b *Barrier) Wait(ctx context.Context) error {
	b.mu.Lock()
	gen := b.generation
	ch := b.notify
	b.arrived++
	if b.arrived >= b.target {
		close(b.notify)
		b.generation++
		b.arrived = 0
		b.notify = make(chan struct{})
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()
	select {
	case <-ctx.Done():
		b.mu.Lock()
		if b.generation == gen && b.arrived > 0 {
			b.arrived--
		}
		b.mu.Unlock()
		return ctx.Err()
	case <-ch:
		return nil
	}
}
func (b *Barrier) Target() int { b.mu.Lock(); defer b.mu.Unlock(); return b.target }

type Semaphore struct{ tokens chan struct{} }

func NewSemaphore(limit int) (*Semaphore, error) {
	if limit < 1 {
		return nil, fmt.Errorf("limit must be positive")
	}
	return &Semaphore{tokens: make(chan struct{}, limit)}, nil
}
func (s *Semaphore) Acquire(ctx context.Context) error {
	select {
	case s.tokens <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.tokens <- struct{}{}:
		return true
	default:
		return false
	}
}
func (s *Semaphore) Release() bool {
	select {
	case <-s.tokens:
		return true
	default:
		return false
	}
}
func (s *Semaphore) InUse() int { return len(s.tokens) }
func (s *Semaphore) Limit() int { return cap(s.tokens) }

type RateLimitBucket struct {
	mu              sync.Mutex
	Capacity        float64
	Tokens          float64
	RefillPerSecond float64
	last            time.Time
}

func NewRateLimitBucket(capacity, refillPerSecond float64) *RateLimitBucket {
	if capacity < 0 {
		capacity = 0
	}
	if refillPerSecond < 0 {
		refillPerSecond = 0
	}
	return &RateLimitBucket{Capacity: capacity, Tokens: capacity, RefillPerSecond: refillPerSecond, last: time.Now()}
}
func (b *RateLimitBucket) refill(now time.Time) {
	if now.Before(b.last) {
		b.last = now
		return
	}
	b.Tokens += now.Sub(b.last).Seconds() * b.RefillPerSecond
	if b.Tokens > b.Capacity {
		b.Tokens = b.Capacity
	}
	b.last = now
}
func (b *RateLimitBucket) Allow(cost float64, now time.Time) bool {
	if cost <= 0 {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill(now)
	if b.Tokens < cost {
		return false
	}
	b.Tokens -= cost
	return true
}
func (b *RateLimitBucket) Remaining(now time.Time) float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill(now)
	return b.Tokens
}
