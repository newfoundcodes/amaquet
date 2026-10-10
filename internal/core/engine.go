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

package core

import (
	"container/heap"
	"context"
	"encoding/base64"
	"encoding/json"
	"hash/fnv"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/newfoundcodes/amaquet/internal/compression"
)

type shard struct {
	mu sync.RWMutex
	m  map[string]*Entry
}

type expiryItem struct {
	key       string
	expiresAt time.Time
	version   uint64
	index     int
}

type expiryHeap []*expiryItem

type Engine struct {
	shards     []shard
	closed     atomic.Bool
	stop       chan struct{}
	wakeExpiry chan struct{}
	wg         sync.WaitGroup

	keyCount    atomic.Int64
	expMu       sync.Mutex
	expiries    expiryHeap
	expiryByKey map[string]*expiryItem

	statsMu        sync.RWMutex
	compressedKeys int
	originalBytes  int64
	storedBytes    int64
	byAlgorithm    map[string]int

	memoryBytes    atomic.Int64
	reservedBytes  atomic.Int64
	maxMemory      atomic.Int64
	capacityMu     sync.Mutex
	evictionMu     sync.Mutex
	evictionPolicy string
	evicted        atomic.Uint64
	evictionCursor atomic.Uint64
}

type MemoryStats struct {
	UsedBytes      int64  `json:"used_bytes"`
	HeapAllocBytes uint64 `json:"heap_alloc_bytes"`
	HeapSysBytes   uint64 `json:"heap_sys_bytes"`
	ReservedBytes  int64  `json:"reserved_bytes"`
	MaxBytes       int64  `json:"max_bytes"`
	Policy         string `json:"policy"`
	Evicted        uint64 `json:"evicted_keys"`
}

type CompressionStats struct {
	CompressedKeys int            `json:"compressed_keys"`
	OriginalBytes  int64          `json:"original_bytes"`
	StoredBytes    int64          `json:"stored_bytes"`
	SavingsBytes   int64          `json:"savings_bytes"`
	SavingsPercent float64        `json:"savings_percent"`
	ByAlgorithm    map[string]int `json:"by_algorithm"`
}

const defaultShardCount = 256

func (h expiryHeap) Len() int {
	return len(h)
}

func (h expiryHeap) Less(i, j int) bool {
	return h[i].expiresAt.Before(h[j].expiresAt)
}

func (h expiryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *expiryHeap) Push(x any) {
	it := x.(*expiryItem)
	it.index = len(*h)
	*h = append(*h, it)
}

func (h *expiryHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]

	old[n-1] = nil
	it.index = -1
	*h = old[:n-1]

	return it
}

func NewEngine() *Engine {
	e := &Engine{
		shards:         make([]shard, defaultShardCount),
		stop:           make(chan struct{}),
		wakeExpiry:     make(chan struct{}, 1),
		byAlgorithm:    map[string]int{},
		expiryByKey:    map[string]*expiryItem{},
		evictionPolicy: "noeviction",
	}

	for i := range e.shards {
		e.shards[i].m = make(map[string]*Entry)
	}

	heap.Init(&e.expiries)
	e.wg.Add(1)

	go e.expirer()
	return e
}

func (e *Engine) shardFor(key string) *shard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))

	return &e.shards[int(h.Sum32())%len(e.shards)]
}

func estimateValueSize(v Value) int64 {
	if x, ok := v.Data.(interface{ ApproxSize() int64 }); ok {
		return x.ApproxSize()
	}

	if p, ok := v.Data.(compression.Payload); ok {
		return int64(len(p.Data)) + 64
	}

	if b, ok := v.Data.([]byte); ok {
		return int64(len(b)) + 32
	}

	if str, ok := v.Data.(string); ok {
		return int64(len(str)) + 32
	}

	if snap, ok := v.Data.(interface{ Snapshot() any }); ok {
		if b, err := json.Marshal(snap.Snapshot()); err == nil {
			return int64(len(b)) + 64
		}
	}

	if b, err := json.Marshal(v.Data); err == nil {
		return int64(len(b)) + 64
	}

	return 256
}

func (e *Engine) ConfigureMemory(maxBytes int64, policy string) error {
	if maxBytes < 0 {
		return ErrInvalidArgument
	}

	switch policy {
	case "", "noeviction", "allkeys-lru", "allkeys-lfu", "volatile-ttl":
	default:
		return ErrInvalidArgument
	}

	if policy == "" {
		policy = "noeviction"
	}

	e.maxMemory.Store(maxBytes)
	if maxBytes > 0 {
		debug.SetMemoryLimit(maxBytes + maxBytes/4)
	} else {
		debug.SetMemoryLimit(-1)
	}

	e.evictionMu.Lock()
	e.evictionPolicy = policy
	e.evictionMu.Unlock()

	return nil
}

func (e *Engine) MemoryUsage() int64 {
	n := e.memoryBytes.Load()
	if n < 0 {
		return 0
	}

	return n
}

func (e *Engine) MemoryStats() MemoryStats {
	e.evictionMu.Lock()
	p := e.evictionPolicy
	e.evictionMu.Unlock()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	return MemoryStats{
		UsedBytes:      e.MemoryUsage(),
		HeapAllocBytes: ms.HeapAlloc,
		HeapSysBytes:   ms.HeapSys,
		ReservedBytes:  e.reservedBytes.Load(),
		MaxBytes:       e.maxMemory.Load(),
		Policy:         p,
		Evicted:        e.evicted.Load(),
	}
}

func (e *Engine) ReserveCapacity(need int64, exclude string) (func(), error) {
	if need <= 0 || e.maxMemory.Load() <= 0 {
		return func() {}, nil
	}

	e.capacityMu.Lock()
	max := e.maxMemory.Load()

	if need > max {
		e.capacityMu.Unlock()
		return nil, ErrMemoryLimit
	}

	for e.MemoryUsage()+e.reservedBytes.Load()+need > max {
		e.evictionMu.Lock()
		policy := e.evictionPolicy
		e.evictionMu.Unlock()

		if policy == "noeviction" || !e.evictOne(policy, exclude) {
			e.capacityMu.Unlock()
			return nil, ErrMemoryLimit
		}
	}

	e.reservedBytes.Add(need)
	e.capacityMu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			e.reservedBytes.Add(-need)
		})
	}, nil
}

func (e *Engine) evictOne(policy, exclude string) bool {
	type candidate struct {
		key  string
		last int64
		ttl  time.Duration
		freq uint64
	}

	var best *candidate
	now := time.Now()

	const shardSamples = defaultShardCount
	const entriesPerShard = 2

	start := int(e.evictionCursor.Add(shardSamples)-shardSamples) % len(e.shards)
	for n := 0; n < shardSamples; n++ {
		i := (start + n) % len(e.shards)
		s := &e.shards[i]
		s.mu.RLock()

		seen := 0
		for k, ent := range s.m {
			if k == exclude || ent.Expired(now) {
				continue
			}
			seen++

			c := candidate{key: k, last: ent.LastAccess().UnixNano(), freq: ent.AccessCount(), ttl: time.Duration(1<<63 - 1)}
			if !ent.ExpiresAt.IsZero() {
				c.ttl = time.Until(ent.ExpiresAt)
			}

			choose := false
			switch policy {
			case "allkeys-lru":
				choose = best == nil || c.last < best.last
			case "allkeys-lfu":
				choose = best == nil || c.freq < best.freq
			case "volatile-ttl":
				if ent.ExpiresAt.IsZero() {
					continue
				}
				choose = best == nil || c.ttl < best.ttl
			}

			if choose {
				x := c
				best = &x
			}

			if seen >= entriesPerShard {
				break
			}
		}

		s.mu.RUnlock()
	}

	if best == nil {
		return false
	}

	if e.Delete(best.key) > 0 {
		e.evicted.Add(1)
		return true
	}

	return false
}

func (e *Engine) ReserveCapacityForMutation(key string, estimatedGrowth int64) (func(), error) {
	return e.ReserveCapacity(estimatedGrowth, key)
}

func (e *Engine) EnsureCapacityForMutation(key string, estimatedGrowth int64) error {
	release, err := e.ReserveCapacityForMutation(key, estimatedGrowth)
	if err != nil {
		return err
	}

	release()
	return nil
}

func compressionInfo(v Value) (algorithm string, original, stored int64, ok bool) {
	p, ok := v.Data.(compression.Payload)
	if !ok {
		return "", 0, 0, false
	}

	return p.Algorithm, int64(p.OriginalSize), int64(len(p.Data)), true
}

func (e *Engine) addStats(v Value) {
	alg, orig, stored, ok := compressionInfo(v)
	if !ok {
		return
	}
	e.statsMu.Lock()

	e.compressedKeys++
	e.originalBytes += orig
	e.storedBytes += stored

	e.byAlgorithm[alg]++
	e.statsMu.Unlock()
}

func (e *Engine) removeStats(v Value) {
	alg, orig, stored, ok := compressionInfo(v)
	if !ok {
		return
	}

	e.statsMu.Lock()
	e.compressedKeys--

	e.originalBytes -= orig
	e.storedBytes -= stored
	e.byAlgorithm[alg]--

	if e.byAlgorithm[alg] <= 0 {
		delete(e.byAlgorithm, alg)
	}
	e.statsMu.Unlock()
}

func (e *Engine) replaceStats(old, new Value) {
	e.removeStats(old)
	e.addStats(new)
}

func (e *Engine) scheduleExpiry(key string, ent *Entry) {
	if ent.ExpiresAt.IsZero() {
		e.unscheduleExpiry(key)
		return
	}

	e.expMu.Lock()
	if old := e.expiryByKey[key]; old != nil {
		if old.version > ent.Version {
			e.expMu.Unlock()
			return
		}

		old.expiresAt = ent.ExpiresAt
		old.version = ent.Version

		heap.Fix(&e.expiries, old.index)
	} else {
		it := &expiryItem{
			key:       key,
			expiresAt: ent.ExpiresAt,
			version:   ent.Version,
		}

		heap.Push(&e.expiries, it)
		e.expiryByKey[key] = it
	}
	e.expMu.Unlock()

	select {
	case e.wakeExpiry <- struct{}{}:
	default:
	}
}

func (e *Engine) unscheduleExpiry(key string) {
	e.expMu.Lock()

	if it := e.expiryByKey[key]; it != nil {
		if it.index >= 0 && it.index < len(e.expiries) {
			heap.Remove(&e.expiries, it.index)
		}

		delete(e.expiryByKey, key)
	}

	e.expMu.Unlock()
}

func (e *Engine) deleteLocked(s *shard, key string, ent *Entry) {
	delete(s.m, key)

	e.keyCount.Add(-1)
	e.removeStats(ent.Value)

	e.memoryBytes.Add(-ent.sizeBytes)
	e.unscheduleExpiry(key)
}

func (e *Engine) purgeExpiredLocked(s *shard, key string, now time.Time) bool {
	ent, ok := s.m[key]
	if ok && ent.Expired(now) {
		e.deleteLocked(s, key, ent)
		return true
	}

	return false
}

func (e *Engine) Set(key string, v Value, ttl time.Duration, nx, xx bool) error {
	return e.setWithReservation(key, v, ttl, nx, xx, 0)
}

func (e *Engine) SetReserved(key string, v Value, ttl time.Duration, nx, xx bool, reservedGrowth int64) error {
	if reservedGrowth < 0 {
		return ErrInvalidArgument
	}

	return e.setWithReservation(key, v, ttl, nx, xx, reservedGrowth)
}

func (e *Engine) setWithReservation(key string, v Value, ttl time.Duration, nx, xx bool, externallyReserved int64) error {
	if key == "" {
		return ErrInvalidArgument
	}

	size := int64(len(key)) + estimateValueSize(v) + 96
	s := e.shardFor(key)

	for {
		now := time.Now().UTC()

		var oldSize int64
		s.mu.RLock()

		if cur := s.m[key]; cur != nil && !cur.Expired(now) {
			oldSize = cur.sizeBytes
		}

		s.mu.RUnlock()
		need := size - oldSize

		if need < 0 {
			need = 0
		}

		var release func()
		if externallyReserved > 0 {
			if need > externallyReserved {
				return ErrMemoryLimit
			}

			release = func() {}
		} else {
			var err error
			release, err = e.ReserveCapacity(need, key)

			if err != nil {
				return err
			}
		}
		s.mu.Lock()

		cur, exists := s.m[key]
		if exists && cur.Expired(now) {
			e.deleteLocked(s, key, cur)

			exists = false
			cur = nil
		}

		if nx && exists {
			s.mu.Unlock()
			release()

			return ErrExists
		}

		if xx && !exists {
			s.mu.Unlock()
			release()

			return ErrNotFound
		}

		actualOld := int64(0)
		if exists {
			actualOld = cur.sizeBytes
		}

		actualNeed := size - actualOld
		if actualNeed < 0 {
			actualNeed = 0
		}

		if externallyReserved > 0 {
			if actualNeed > externallyReserved {
				s.mu.Unlock()
				return ErrMemoryLimit
			}
		} else if actualNeed > need {
			s.mu.Unlock()
			release()
			continue
		}

		ver := uint64(1)
		created := now

		if exists {
			ver = cur.Version + 1
			created = cur.CreatedAt

			e.replaceStats(cur.Value, v)
			e.memoryBytes.Add(size - cur.sizeBytes)
		} else {
			e.keyCount.Add(1)
			e.addStats(v)
			e.memoryBytes.Add(size)
		}

		ent := &Entry{
			Value:     v,
			CreatedAt: created,
			UpdatedAt: now,
			Version:   ver,
			sizeBytes: size,
		}
		ent.lastAccess.Store(now.UnixNano())

		if ttl > 0 {
			ent.ExpiresAt = now.Add(ttl)
		}

		s.m[key] = ent
		e.scheduleExpiry(key, ent)
		s.mu.Unlock()

		release()
		return nil
	}
}

func (e *Engine) SetAbsolute(key string, v Value, expiresAt time.Time, nx, xx bool) error {
	now := time.Now().UTC()
	ttl := time.Duration(0)

	if !expiresAt.IsZero() {
		if !expiresAt.After(now) {
			e.Delete(key)
			return nil
		}

		ttl = expiresAt.Sub(now)
	}

	return e.Set(key, v, ttl, nx, xx)
}

func (e *Engine) SetAbsoluteReserved(key string, v Value, expiresAt time.Time, nx, xx bool, reservedGrowth int64) error {
	now := time.Now().UTC()
	ttl := time.Duration(0)

	if !expiresAt.IsZero() {
		if !expiresAt.After(now) {
			e.Delete(key)
			return nil
		}

		ttl = expiresAt.Sub(now)
	}

	return e.SetReserved(key, v, ttl, nx, xx, reservedGrowth)
}

func (e *Engine) CompareAndSet(key string, expectedVersion uint64, v Value, ttl time.Duration) error {
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().UTC().Add(ttl)
	}

	return e.CompareAndSetAbsolute(key, expectedVersion, v, expiresAt)
}

func (e *Engine) CompareAndSetAbsolute(key string, expectedVersion uint64, v Value, expiresAt time.Time) error {
	now := time.Now().UTC()
	s := e.shardFor(key)
	s.mu.RLock()

	cur := s.m[key]
	if cur == nil || cur.Expired(now) {
		s.mu.RUnlock()
		if cur != nil {
			s.mu.Lock()
			e.purgeExpiredLocked(s, key, now)
			s.mu.Unlock()
		}

		return ErrNotFound
	}

	if cur.Version != expectedVersion {
		s.mu.RUnlock()
		return ErrConflict
	}

	oldSize := cur.sizeBytes
	s.mu.RUnlock()

	if !expiresAt.IsZero() && !expiresAt.After(now) {
		s.mu.Lock()

		cur = s.m[key]
		if cur == nil || cur.Expired(now) {
			if cur != nil {
				e.deleteLocked(s, key, cur)
			}

			s.mu.Unlock()
			return ErrNotFound
		}

		if cur.Version != expectedVersion {
			s.mu.Unlock()
			return ErrConflict
		}

		e.deleteLocked(s, key, cur)
		s.mu.Unlock()

		return nil
	}

	size := int64(len(key)) + estimateValueSize(v) + 96
	need := size - oldSize

	if need < 0 {
		need = 0
	}

	release, err := e.ReserveCapacity(need, key)
	if err != nil {
		return err
	}

	defer release()
	s.mu.Lock()

	cur = s.m[key]
	if cur == nil || cur.Expired(now) {
		if cur != nil {
			e.deleteLocked(s, key, cur)
		}

		s.mu.Unlock()
		return ErrNotFound
	}

	if cur.Version != expectedVersion {
		s.mu.Unlock()
		return ErrConflict
	}

	n := &Entry{
		Value:     v,
		CreatedAt: cur.CreatedAt,
		UpdatedAt: now,
		Version:   cur.Version + 1,
		sizeBytes: size,
		ExpiresAt: expiresAt,
	}
	n.lastAccess.Store(now.UnixNano())

	e.replaceStats(cur.Value, v)
	e.memoryBytes.Add(size - cur.sizeBytes)

	s.m[key] = n
	e.scheduleExpiry(key, n)

	s.mu.Unlock()
	return nil
}

func (e *Engine) Get(key string) (Value, uint64, error) {
	s := e.shardFor(key)
	now := time.Now()
	s.mu.RLock()

	ent, ok := s.m[key]
	if ok && !ent.Expired(now) {
		v, ver := ent.Value, ent.Version
		ent.Touch()
		s.mu.RUnlock()

		return v, ver, nil
	}
	s.mu.RUnlock()

	if ok {
		s.mu.Lock()
		e.purgeExpiredLocked(s, key, now)
		s.mu.Unlock()
	}

	return Value{}, 0, ErrNotFound
}

func (e *Engine) GetEntry(key string) (*Entry, error) {
	s := e.shardFor(key)
	now := time.Now()
	s.mu.RLock()

	ent, ok := s.m[key]
	if !ok || ent.Expired(now) {
		s.mu.RUnlock()

		if ok {
			s.mu.Lock()
			e.purgeExpiredLocked(s, key, now)
			s.mu.Unlock()
		}

		return nil, ErrNotFound
	}

	cp := &Entry{
		Value:     ent.Value,
		CreatedAt: ent.CreatedAt,
		UpdatedAt: ent.UpdatedAt,
		ExpiresAt: ent.ExpiresAt,
		Version:   ent.Version,
		sizeBytes: ent.sizeBytes,
	}
	cp.accesses.Store(ent.accesses.Load())
	cp.lastAccess.Store(ent.lastAccess.Load())

	s.mu.RUnlock()
	return cp, nil
}

func (e *Engine) Delete(keys ...string) int {
	n := 0
	for _, key := range keys {
		s := e.shardFor(key)
		s.mu.Lock()

		if ent, ok := s.m[key]; ok {
			e.deleteLocked(s, key, ent)
			n++
		}

		s.mu.Unlock()
	}

	return n
}

func (e *Engine) Exists(key string) bool { _, _, err := e.Get(key); return err == nil }

func (e *Engine) Type(key string) (DataType, error) { v, _, err := e.Get(key); return v.Type, err }

func (e *Engine) Expire(key string, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidArgument
	}

	now := time.Now().UTC()
	s := e.shardFor(key)
	s.mu.Lock()

	ent, ok := s.m[key]
	if !ok || ent.Expired(now) {
		if ok {
			e.deleteLocked(s, key, ent)
		}

		s.mu.Unlock()
		return ErrNotFound
	}

	ent.ExpiresAt = now.Add(ttl)
	ent.UpdatedAt = now
	ent.Version++

	e.scheduleExpiry(key, ent)
	s.mu.Unlock()

	return nil
}

func (e *Engine) ExpireAt(key string, at time.Time) error {
	now := time.Now().UTC()
	if !at.After(now) {
		if e.Delete(key) == 0 {
			return ErrNotFound
		}

		return nil
	}

	return e.Expire(key, time.Until(at))
}

func (e *Engine) Persist(key string) error {
	now := time.Now().UTC()
	s := e.shardFor(key)
	s.mu.Lock()

	ent, ok := s.m[key]
	if !ok || ent.Expired(now) {
		if ok {
			e.deleteLocked(s, key, ent)
		}

		s.mu.Unlock()
		return ErrNotFound
	}

	ent.ExpiresAt = time.Time{}
	ent.UpdatedAt = now
	ent.Version++

	e.unscheduleExpiry(key)
	s.mu.Unlock()

	return nil
}

func (e *Engine) TTL(key string) (time.Duration, error) {
	ent, err := e.GetEntry(key)
	if err != nil {
		return 0, err
	}

	if ent.ExpiresAt.IsZero() {
		return -1, nil
	}

	d := time.Until(ent.ExpiresAt)
	if d < 0 {
		return 0, ErrNotFound
	}

	return d, nil
}

func (e *Engine) Keys(prefix string, limit int) []string {
	if limit <= 0 {
		limit = 1000
	}

	if limit > 10000 {
		limit = 10000
	}

	out := make([]string, 0, min(limit, 1024))
	cursor := ""

	for len(out) < limit {
		page, next := e.Scan(cursor, prefix, limit-len(out))
		out = append(out, page...)

		if next == "" || next == cursor {
			break
		}

		cursor = next
	}

	return out
}

func (e *Engine) Scan(cursor, prefix string, count int) ([]string, string) {
	if count <= 0 {
		count = 100
	}

	if count > 10000 {
		count = 10000
	}

	startShard, lastKey := 0, ""
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err == nil {
			parts := strings.SplitN(string(raw), "\x00", 2)
			if len(parts) == 2 {
				if n, err := strconv.Atoi(parts[0]); err == nil && n >= 0 && n < len(e.shards) {
					startShard, lastKey = n, parts[1]
				}
			}
		}
	}

	out := make([]string, 0, count)
	now := time.Now()

	for si := startShard; si < len(e.shards); si++ {
		s := &e.shards[si]
		s.mu.RLock()

		keys := make([]string, 0, min(len(s.m), count*2))
		for k, ent := range s.m {
			if ent.Expired(now) || (prefix != "" && !strings.HasPrefix(k, prefix)) {
				continue
			}

			keys = append(keys, k)
		}

		s.mu.RUnlock()
		sort.Strings(keys)

		pos := 0
		if si == startShard && lastKey != "" {
			pos = sort.Search(len(keys), func(i int) bool { return keys[i] > lastKey })
		}

		for ; pos < len(keys); pos++ {
			out = append(out, keys[pos])
			if len(out) == count {
				nextRaw := strconv.Itoa(si) + "\x00" + keys[pos]
				return out, base64.RawURLEncoding.EncodeToString([]byte(nextRaw))
			}
		}

		lastKey = ""
	}

	return out, ""
}

func (e *Engine) Len() int {
	n := e.keyCount.Load()
	if n < 0 {
		return 0
	}

	return int(n)
}

func (e *Engine) CompressionStats() CompressionStats {
	e.statsMu.RLock()

	st := CompressionStats{
		CompressedKeys: e.compressedKeys,
		OriginalBytes:  e.originalBytes,
		StoredBytes:    e.storedBytes,
		ByAlgorithm:    make(map[string]int, len(e.byAlgorithm)),
	}

	for k, v := range e.byAlgorithm {
		st.ByAlgorithm[k] = v
	}
	e.statsMu.RUnlock()

	st.SavingsBytes = st.OriginalBytes - st.StoredBytes
	if st.OriginalBytes > 0 {
		st.SavingsPercent = 100 * float64(st.SavingsBytes) / float64(st.OriginalBytes)
	}

	return st
}

func (e *Engine) Update(key string, expected DataType, fn func(Value) (Value, error)) (Value, error) {
	return e.updateWithReservation(key, expected, 0, fn)
}

func (e *Engine) UpdateReserved(key string, expected DataType, reservedGrowth int64, fn func(Value) (Value, error)) (Value, error) {
	if reservedGrowth < 0 {
		return Value{}, ErrInvalidArgument
	}

	return e.updateWithReservation(key, expected, reservedGrowth, fn)
}

func (e *Engine) updateWithReservation(key string, expected DataType, externallyReserved int64, fn func(Value) (Value, error)) (Value, error) {
	s := e.shardFor(key)
	for {
		now := time.Now().UTC()
		s.mu.Lock()

		ent, ok := s.m[key]
		if !ok || ent.Expired(now) {
			if ok {
				e.deleteLocked(s, key, ent)
			}

			s.mu.Unlock()
			return Value{}, ErrNotFound
		}

		if expected != TypeInvalid && ent.Value.Type != expected {
			s.mu.Unlock()
			return Value{}, ErrWrongType
		}

		version := ent.Version
		oldSize := ent.sizeBytes
		oldValue := ent.Value
		nv, err := fn(oldValue)

		if err != nil {
			s.mu.Unlock()
			return Value{}, err
		}

		size := int64(len(key)) + estimateValueSize(nv) + 96
		delta := size - oldSize

		if delta < 0 {
			delta = 0
		}

		if externallyReserved > 0 {
			if delta > externallyReserved {
				s.mu.Unlock()
				return Value{}, ErrMemoryLimit
			}
		} else if delta > 0 {
			s.mu.Unlock()

			release, err := e.ReserveCapacity(delta, key)
			if err != nil {
				return Value{}, err
			}
			s.mu.Lock()

			cur := s.m[key]
			if cur == nil || cur.Expired(time.Now().UTC()) || cur.Version != version {
				s.mu.Unlock()
				release()
				continue
			}

			nv, err = fn(cur.Value)
			if err != nil {
				s.mu.Unlock()
				release()
				return Value{}, err
			}

			size = int64(len(key)) + estimateValueSize(nv) + 96
			actual := size - cur.sizeBytes

			if actual > delta {
				s.mu.Unlock()
				release()
				continue
			}

			oldValue = cur.Value
			oldSize = cur.sizeBytes

			e.replaceStats(oldValue, nv)
			e.memoryBytes.Add(size - oldSize)

			cur.sizeBytes = size
			cur.Value = nv
			cur.UpdatedAt = time.Now().UTC()
			cur.Version++

			s.mu.Unlock()
			release()

			return nv, nil
		}

		e.replaceStats(oldValue, nv)
		e.memoryBytes.Add(size - oldSize)

		ent.sizeBytes = size
		ent.Value = nv
		ent.UpdatedAt = now
		ent.Version++

		s.mu.Unlock()
		return nv, nil
	}
}

func (e *Engine) expirer() {
	defer e.wg.Done()
	timer := time.NewTimer(time.Hour)

	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	for {
		e.expMu.Lock()
		if len(e.expiries) == 0 {
			e.expMu.Unlock()
			select {
			case <-e.wakeExpiry:
				continue
			case <-e.stop:
				return
			}
		}

		now := time.Now()
		next := e.expiries[0]
		wait := time.Until(next.expiresAt)

		if wait > 0 {
			e.expMu.Unlock()
			timer.Reset(wait)

			select {
			case <-timer.C:
				continue
			case <-e.wakeExpiry:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				continue
			case <-e.stop:
				return
			}
		}

		popped := heap.Pop(&e.expiries).(*expiryItem)
		if cur := e.expiryByKey[popped.key]; cur == popped {
			delete(e.expiryByKey, popped.key)
		}
		e.expMu.Unlock()

		s := e.shardFor(next.key)
		s.mu.Lock()

		if ent, ok := s.m[next.key]; ok && ent.Version == next.version && !ent.ExpiresAt.IsZero() && !now.Before(ent.ExpiresAt) {
			e.deleteLocked(s, next.key, ent)
		}
		s.mu.Unlock()
	}
}

func (e *Engine) Close() error {
	if e.closed.CompareAndSwap(false, true) {
		close(e.stop)
		e.wg.Wait()
	}

	return nil
}

func (e *Engine) Transaction(ctx context.Context, keys []string, fn func(map[string]*Entry) error) error {
	ids := make([]int, 0, len(keys))
	seen := map[int]bool{}

	for _, k := range keys {
		h := fnv.New32a()
		_, _ = h.Write([]byte(k))

		id := int(h.Sum32()) % len(e.shards)
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	sort.Ints(ids)
	for _, id := range ids {
		e.shards[id].mu.Lock()
	}

	defer func() {
		for i := len(ids) - 1; i >= 0; i-- {
			e.shards[ids[i]].mu.Unlock()
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	now := time.Now()
	view := make(map[string]*Entry, len(keys))
	original := make(map[string]*Entry, len(keys))

	for _, k := range keys {
		if v, ok := e.shardFor(k).m[k]; ok && !v.Expired(now) {
			rv := reflect.ValueOf(v.Value.Data)
			if rv.IsValid() {
				switch rv.Kind() {
				case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
					return ErrInvalidArgument
				}
			}

			cp := &Entry{
				Value:     v.Value,
				CreatedAt: v.CreatedAt,
				UpdatedAt: v.UpdatedAt,
				ExpiresAt: v.ExpiresAt,
				Version:   v.Version,
				sizeBytes: v.sizeBytes,
			}

			cp.accesses.Store(v.accesses.Load())
			cp.lastAccess.Store(v.lastAccess.Load())

			view[k] = cp
			original[k] = v
		}
	}

	if err := fn(view); err != nil {
		return err
	}

	var delta int64
	for k, staged := range view {
		if staged == nil {
			continue
		}

		staged.sizeBytes = int64(len(k)) + estimateValueSize(staged.Value) + 96
		if old := original[k]; old != nil {
			delta += staged.sizeBytes - old.sizeBytes
		} else {
			delta += staged.sizeBytes
		}
	}

	if max := e.maxMemory.Load(); max > 0 && delta > 0 && e.MemoryUsage()+delta > max {
		return ErrMemoryLimit
	}

	for _, k := range keys {
		s := e.shardFor(k)
		old := original[k]
		staged, exists := view[k]

		if old != nil && (!exists || staged == nil) {
			e.deleteLocked(s, k, old)
			continue
		}

		if staged == nil {
			continue
		}

		if old != nil {
			e.replaceStats(old.Value, staged.Value)
			e.memoryBytes.Add(staged.sizeBytes - old.sizeBytes)

			if staged.Version <= old.Version {
				staged.Version = old.Version + 1
			}
		} else {
			e.keyCount.Add(1)
			e.addStats(staged.Value)
			e.memoryBytes.Add(staged.sizeBytes)

			if staged.Version == 0 {
				staged.Version = 1
			}
		}

		staged.UpdatedAt = time.Now().UTC()
		if staged.CreatedAt.IsZero() {
			staged.CreatedAt = staged.UpdatedAt
		}

		staged.lastAccess.Store(staged.UpdatedAt.UnixNano())
		s.m[k] = staged

		if staged.ExpiresAt.IsZero() {
			e.unscheduleExpiry(k)
		} else {
			e.scheduleExpiry(k, staged)
		}
	}

	return nil
}

func (e *Engine) BumpVersion(key string) error {
	return e.BumpVersionReserved(key, 0)
}

func (e *Engine) BumpVersionReserved(key string, reservedGrowth int64) error {
	if reservedGrowth < 0 {
		return ErrInvalidArgument
	}

	s := e.shardFor(key)
	now := time.Now().UTC()
	s.mu.Lock()

	ent, ok := s.m[key]
	if !ok || ent.Expired(now) {
		if ok {
			e.deleteLocked(s, key, ent)
		}

		s.mu.Unlock()
		return ErrNotFound
	}

	size := int64(len(key)) + estimateValueSize(ent.Value) + 96
	delta := size - ent.sizeBytes

	if delta > 0 && reservedGrowth > 0 && delta > reservedGrowth {
		s.mu.Unlock()
		return ErrMemoryLimit
	}

	if delta > 0 && reservedGrowth == 0 {
		s.mu.Unlock()

		release, err := e.ReserveCapacity(delta, key)
		if err != nil {
			return err
		}

		defer release()
		s.mu.Lock()

		ent, ok = s.m[key]
		if !ok || ent.Expired(time.Now().UTC()) {
			if ok {
				e.deleteLocked(s, key, ent)
			}

			s.mu.Unlock()
			return ErrNotFound
		}

		size = int64(len(key)) + estimateValueSize(ent.Value) + 96
		delta = size - ent.sizeBytes

		if delta > 0 && e.maxMemory.Load() > 0 && e.MemoryUsage()+delta > e.maxMemory.Load()+e.reservedBytes.Load() {
			s.mu.Unlock()
			return ErrMemoryLimit
		}
	}
	e.memoryBytes.Add(delta)

	ent.sizeBytes = size
	ent.Version++

	ent.UpdatedAt = time.Now().UTC()
	s.mu.Unlock()

	return nil
}
