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
	"encoding/base64"
	"github.com/newfoundcodes/amaquet/internal/core"
	"sort"
	"time"
)

func (l *List) Snapshot() any { return l.Range(0, -1) }
func (a *Array) Snapshot() any {
	return map[string]any{"typed": a.Typed, "element_type": a.ElementType.String(), "items": a.Values()}
}
func (d *Deque) Snapshot() any {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]core.Value, d.size)
	for i := 0; i < d.size; i++ {
		out[i] = d.buf[(d.head+i)%len(d.buf)]
	}
	return out
}
func (r *RingBuffer) Snapshot() any { return r.Values() }
func (t Tuple) Snapshot() any       { return cloneValues(t.Items) }
func (s *Set) Snapshot() any        { return s.Values() }
func (z *SortedSet) Snapshot() any  { return z.Range(0, -1, false) }
func (s *OrderedSet) Snapshot() any { return s.Values() }
func (m *MultiSet) Snapshot() any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	type e struct {
		Value core.Value `json:"value"`
		Count uint64     `json:"count"`
	}
	out := make([]e, 0, len(m.counts))
	for k, c := range m.counts {
		out = append(out, e{m.values[k], c})
	}
	sort.Slice(out, func(i, j int) bool { return valueKey(out[i].Value) < valueKey(out[j].Value) })
	return out
}
func (h *HashMap) Snapshot() any {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := map[string]core.Value{}
	for k, v := range h.items {
		out[k] = v
	}
	return out
}
func (m *OrderedMap) Snapshot() any { return m.Entries() }
func (m *MultiMap) Snapshot() any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string][]core.Value{}
	for k, v := range m.items {
		out[k] = cloneValues(v)
	}
	return out
}
func (j *JSONDocument) Snapshot() any { return j.Value() }
func (m MessagePack) Snapshot() any   { return base64.StdEncoding.EncodeToString(m.Data) }
func (c CBOR) Snapshot() any          { return base64.StdEncoding.EncodeToString(c.Data) }
func (r *Record) Snapshot() any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := map[string]any{"schema": r.Schema, "fields": map[string]core.Value{}}
	f := out["fields"].(map[string]core.Value)
	for k, v := range r.Fields {
		f[k] = v
	}
	return out
}
func (t *Tensor) Snapshot() any {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return map[string]any{"shape": append([]int(nil), t.Shape...), "dtype": t.DType, "data": append([]float64(nil), t.Data...)}
}
func (q *FIFOQueue) Snapshot() any { q.mu.Lock(); defer q.mu.Unlock(); return cloneValues(q.items) }
func (s *LIFOStack) Snapshot() any { s.mu.Lock(); defer s.mu.Unlock(); return cloneValues(s.items) }
func (q *PriorityQueue) Snapshot() any {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]PriorityItem, len(q.h))
	for i, p := range q.h {
		out[i] = *p
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Priority > out[j].Priority })
	return out
}
func (q *BlockingQueue) Snapshot() any {
	q.mu.Lock()
	defer q.mu.Unlock()
	return map[string]any{"closed": q.closed, "items": cloneValues(q.items)}
}
func (q *DelayedQueue) Snapshot() any {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]DelayedItem(nil), q.items...)
}
func (q *ReliableQueue) Snapshot() any {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.requeueExpiredLocked(time.Now())
	return map[string]any{"ready": append([]ReliableMessage(nil), q.ready...), "pending": q.pending, "dead": append([]ReliableMessage(nil), q.dead...)}
}
func (s *Stream) Snapshot() any {
	return map[string]any{"length": s.Len(), "entries": s.Range(nil, nil, 1000)}
}
func (g *ConsumerGroup) Snapshot() any {
	g.mu.Lock()
	defer g.mu.Unlock()
	p := make([]PendingMessage, 0, len(g.Pending))
	for _, v := range g.Pending {
		p = append(p, v)
	}
	return map[string]any{"name": g.Name, "last_delivered": g.LastDelivered, "pending": p}
}
func (p *PubSub) Snapshot() any {
	p.mu.RLock()
	defer p.mu.RUnlock()
	m := map[string]int{}
	for k, v := range p.channels {
		m[k] = len(v)
	}
	return m
}
func (t *PersistentTopic) Snapshot() any {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return append([]TopicMessage(nil), t.messages...)
}
func (l *EventLog) Snapshot() any { return l.Read(1, 1000) }
func (b *Bitmap) Snapshot() any {
	return map[string]any{"count": b.Count(), "bytes": base64.StdEncoding.EncodeToString(b.Bytes())}
}
func (b *BitField) Snapshot() any {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return base64.StdEncoding.EncodeToString(append([]byte(nil), b.data...))
}
func (r *RoaringBitmap) Snapshot() any { return r.Values() }
func (h *HyperLogLog) Snapshot() any {
	return map[string]any{"precision": h.Precision, "count": h.Count()}
}
func (b *BloomFilter) Snapshot() any         { return map[string]any{"bits": b.M, "hashes": b.K} }
func (b *CountingBloomFilter) Snapshot() any { return map[string]any{"counters": b.M, "hashes": b.K} }
func (c *CuckooFilter) Snapshot() any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return map[string]any{"capacity": len(c.buckets) * 4, "count": c.count}
}
func (c *CountMinSketch) Snapshot() any { return map[string]any{"width": c.Width, "depth": c.Depth} }
func (t *TopK) Snapshot() any           { return t.Items() }
func (t *TDigest) Snapshot() any {
	t.mu.Lock()
	defer t.mu.Unlock()
	return map[string]any{"compression": t.Compression, "count": t.total, "centroids": len(t.centroids)}
}
func (s *TimeSeries) Snapshot() any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{"labels": s.Labels, "retention": s.Retention.String(), "points": append([]TimeSeriesPoint(nil), s.points...)}
}
func (s *CounterSeries) Snapshot() any { return s.Series.Snapshot() }
func (s *GaugeSeries) Snapshot() any   { return s.Series.Snapshot() }
func (h *Histogram) SnapshotAny() any {
	b, c, n, sum := h.Snapshot()
	return map[string]any{"bounds": b, "counts": c, "count": n, "sum": sum}
}
func (g *GeoSpatialIndex) Snapshot() any {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]GeoItem, 0, len(g.items))
	for _, v := range g.items {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func (v *VectorSet) Snapshot() any {
	v.mu.RLock()
	defer v.mu.RUnlock()
	out := make([]VectorItem, 0, len(v.items))
	for _, x := range v.items {
		out = append(out, VectorItem{x.ID, x.Vector.Clone(), cloneMetadata(x.Metadata)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return map[string]any{"dim": v.Dim, "metric": v.Metric, "items": out}
}
func (h *HNSWIndex) Snapshot() any {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return map[string]any{"dim": h.Dim, "metric": h.Metric, "m": h.M, "ef_search": h.efSearch, "count": len(h.vectors), "entry": h.entry}
}
func (f *FlatVectorIndex) Snapshot() any { return f.Set.Snapshot() }
func (b *BTreeIndex) Snapshot() any {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return map[string]any{"degree": b.Degree, "keys": len(b.entries)}
}
func (h *HashIndex) Snapshot() any {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return map[string]any{"keys": len(h.items)}
}
func (r *RadixTree) Snapshot() any { return r.Prefix("", 1000) }
func (i *InvertedIndex) Snapshot() any {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return map[string]any{"terms": len(i.terms), "documents": len(i.docTerms)}
}
func (s *SecondaryIndex) Snapshot() any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.fields))
	for k := range s.fields {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func (a *AdjacencySet) Snapshot() any {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := map[string]map[string]float64{}
	for k, v := range a.edges {
		cp := map[string]float64{}
		for x, w := range v {
			cp[x] = w
		}
		out[k] = cp
	}
	return out
}
func (c *AtomicCounter) Snapshot() any { return c.Load() }
func (l *Lease) Snapshot() any {
	l.mu.Lock()
	defer l.mu.Unlock()
	return map[string]any{"owner": l.Owner, "expires_at": l.ExpiresAt}
}
func (l *Lock) Snapshot() any      { return l.lease.Snapshot() }
func (b *Barrier) Snapshot() any   { return map[string]any{"target": b.Target()} }
func (s *Semaphore) Snapshot() any { return map[string]any{"limit": s.Limit(), "in_use": s.InUse()} }
func (b *RateLimitBucket) Snapshot() any {
	return map[string]any{"capacity": b.Capacity, "tokens": b.Remaining(time.Now()), "refill_per_second": b.RefillPerSecond}
}
