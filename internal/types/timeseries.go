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
	"math"
	"sort"
	"sync"
	"time"
)

type TimeSeriesPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}
type TimeSeriesLabels map[string]string

type TimeSeries struct {
	mu        sync.RWMutex
	points    []TimeSeriesPoint
	Labels    TimeSeriesLabels
	Retention time.Duration
}

func NewTimeSeries(labels TimeSeriesLabels, retention time.Duration) *TimeSeries {
	cp := TimeSeriesLabels{}
	for k, v := range labels {
		cp[k] = v
	}
	return &TimeSeries{Labels: cp, Retention: retention}
}
func (s *TimeSeries) Add(ts int64, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := sort.Search(len(s.points), func(i int) bool { return s.points[i].Timestamp >= ts })
	if i < len(s.points) && s.points[i].Timestamp == ts {
		s.points[i].Value = value
	} else {
		s.points = append(s.points, TimeSeriesPoint{})
		copy(s.points[i+1:], s.points[i:])
		s.points[i] = TimeSeriesPoint{ts, value}
	}
	s.trimLocked(time.Now().UnixMilli())
	return nil
}
func (s *TimeSeries) trimLocked(now int64) {
	if s.Retention <= 0 {
		return
	}
	cut := now - s.Retention.Milliseconds()
	i := sort.Search(len(s.points), func(i int) bool { return s.points[i].Timestamp >= cut })
	if i > 0 {
		s.points = append([]TimeSeriesPoint(nil), s.points[i:]...)
	}
}
func (s *TimeSeries) Range(from, to int64) []TimeSeriesPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i := sort.Search(len(s.points), func(i int) bool { return s.points[i].Timestamp >= from })
	j := sort.Search(len(s.points), func(i int) bool { return s.points[i].Timestamp > to })
	return append([]TimeSeriesPoint(nil), s.points[i:j]...)
}
func (s *TimeSeries) Last() (TimeSeriesPoint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.points) == 0 {
		return TimeSeriesPoint{}, false
	}
	return s.points[len(s.points)-1], true
}
func (s *TimeSeries) Len() int { s.mu.RLock(); defer s.mu.RUnlock(); return len(s.points) }

type Aggregation string

const (
	AggAvg   Aggregation = "avg"
	AggSum   Aggregation = "sum"
	AggMin   Aggregation = "min"
	AggMax   Aggregation = "max"
	AggCount Aggregation = "count"
	AggFirst Aggregation = "first"
	AggLast  Aggregation = "last"
)

func (s *TimeSeries) Aggregate(from, to, bucketMillis int64, agg Aggregation) ([]TimeSeriesPoint, error) {
	if bucketMillis <= 0 {
		return nil, fmt.Errorf("bucket must be positive")
	}
	pts := s.Range(from, to)
	if len(pts) == 0 {
		return nil, nil
	}
	type acc struct {
		start         int64
		sum, min, max float64
		count         int
		first, last   float64
	}
	m := map[int64]*acc{}
	keys := []int64{}
	for _, p := range pts {
		b := from + ((p.Timestamp-from)/bucketMillis)*bucketMillis
		a := m[b]
		if a == nil {
			a = &acc{start: b, min: p.Value, max: p.Value, first: p.Value}
			m[b] = a
			keys = append(keys, b)
		}
		a.sum += p.Value
		a.count++
		a.last = p.Value
		if p.Value < a.min {
			a.min = p.Value
		}
		if p.Value > a.max {
			a.max = p.Value
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := make([]TimeSeriesPoint, 0, len(keys))
	for _, k := range keys {
		a := m[k]
		var v float64
		switch agg {
		case AggAvg:
			v = a.sum / float64(a.count)
		case AggSum:
			v = a.sum
		case AggMin:
			v = a.min
		case AggMax:
			v = a.max
		case AggCount:
			v = float64(a.count)
		case AggFirst:
			v = a.first
		case AggLast:
			v = a.last
		default:
			return nil, fmt.Errorf("unknown aggregation %q", agg)
		}
		out = append(out, TimeSeriesPoint{k, v})
	}
	return out, nil
}

type CounterSeries struct{ Series *TimeSeries }

func NewCounterSeries(labels TimeSeriesLabels, retention time.Duration) *CounterSeries {
	return &CounterSeries{NewTimeSeries(labels, retention)}
}
func (c *CounterSeries) Add(ts int64, delta float64) error {
	last, ok := c.Series.Last()
	v := delta
	if ok {
		v = last.Value + delta
	}
	return c.Series.Add(ts, v)
}

type GaugeSeries struct{ Series *TimeSeries }

func NewGaugeSeries(labels TimeSeriesLabels, retention time.Duration) *GaugeSeries {
	return &GaugeSeries{NewTimeSeries(labels, retention)}
}
func (g *GaugeSeries) Set(ts int64, value float64) error { return g.Series.Add(ts, value) }

type Histogram struct {
	mu     sync.RWMutex
	Bounds []float64
	Counts []uint64
	Count  uint64
	Sum    float64
}

func NewHistogram(bounds []float64) *Histogram {
	b := append([]float64(nil), bounds...)
	sort.Float64s(b)
	return &Histogram{Bounds: b, Counts: make([]uint64, len(b)+1)}
}
func (h *Histogram) Observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	i := sort.SearchFloat64s(h.Bounds, v)
	h.Counts[i]++
	h.Count++
	h.Sum += v
}
func (h *Histogram) Snapshot() (bounds []float64, counts []uint64, count uint64, sum float64) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]float64(nil), h.Bounds...), append([]uint64(nil), h.Counts...), h.Count, h.Sum
}
func (h *Histogram) Mean() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.Count == 0 {
		return math.NaN()
	}
	return h.Sum / float64(h.Count)
}

type AggregatedSeries struct {
	Source       *TimeSeries
	BucketMillis int64
	Aggregation  Aggregation
}

func (a *AggregatedSeries) Range(from, to int64) ([]TimeSeriesPoint, error) {
	if a.Source == nil {
		return nil, fmt.Errorf("source required")
	}
	return a.Source.Aggregate(from, to, a.BucketMillis, a.Aggregation)
}
