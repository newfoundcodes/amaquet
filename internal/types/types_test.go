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
	"github.com/newfoundcodes/amaquet/internal/core"
	"math"
	"strings"
	"testing"
	"time"
)

func strv(s string) core.Value { return core.Value{Type: core.TypeUTF8String, Data: UTF8String(s)} }
func TestDecimal(t *testing.T) {
	a, _ := ParseDecimal("10.25")
	b, _ := ParseDecimal("1.5")
	if got := a.Add(b).String(); got != "11.75" {
		t.Fatal(got)
	}
	if got := a.Sub(b).String(); got != "8.75" {
		t.Fatal(got)
	}
}
func TestUUID(t *testing.T) {
	s := "550e8400-e29b-41d4-a716-446655440000"
	u, e := ParseUUID(s)
	if e != nil || u.String() != s {
		t.Fatalf("%v %s", e, u.String())
	}
}
func TestArrayTyped(t *testing.T) {
	a, e := NewArray(core.TypeUTF8String, true)
	if e != nil {
		t.Fatal(e)
	}
	if e = a.Append(strv("x")); e != nil {
		t.Fatal(e)
	}
	if e = a.Append(core.Value{Type: core.TypeInteger, Data: Integer(1)}); e == nil {
		t.Fatal("expected type error")
	}
}
func TestDequeAndRing(t *testing.T) {
	d := NewDeque(2)
	d.PushBack(strv("b"))
	d.PushFront(strv("a"))
	v, _ := d.PopFront()
	if v.Data != UTF8String("a") {
		t.Fatal(v)
	}
	r := NewRingBuffer(2)
	r.Push(strv("a"))
	r.Push(strv("b"))
	r.Push(strv("c"))
	xs := r.Values()
	if len(xs) != 2 || xs[0].Data != UTF8String("b") {
		t.Fatal(xs)
	}
}
func TestSetVariants(t *testing.T) {
	s := NewSet()
	if !s.Add(strv("a")) || s.Add(strv("a")) {
		t.Fatal("set uniqueness")
	}
	o := NewOrderedSet()
	o.Add(strv("b"))
	o.Add(strv("a"))
	if o.Values()[0].Data != UTF8String("b") {
		t.Fatal("order")
	}
	m := NewMultiSet()
	m.Add(strv("a"), 3)
	if m.Remove(strv("a"), 1) != 2 {
		t.Fatal("multiset")
	}
}
func TestSortedSet(t *testing.T) {
	z := NewSortedSet()
	z.Add(strv("b"), 2)
	z.Add(strv("a"), 1)
	r := z.Range(0, -1, false)
	if len(r) != 2 || r[0].Score != 1 {
		t.Fatal(r)
	}
}
func TestMaps(t *testing.T) {
	h := NewHashMap()
	h.Set("x", strv("y"))
	if v, ok := h.Get("x"); !ok || v.Data != UTF8String("y") {
		t.Fatal(v)
	}
	mm := NewMultiMap()
	mm.Add("x", strv("a"))
	mm.Add("x", strv("b"))
	if len(mm.Get("x")) != 2 {
		t.Fatal("multimap")
	}
}
func TestTensor(t *testing.T) {
	x, e := NewTensor([]int{2, 2}, DTypeFloat64, []float64{1, 2, 3, 4})
	if e != nil {
		t.Fatal(e)
	}
	if v, _ := x.Get(1, 0); v != 3 {
		t.Fatal(v)
	}
	if e = x.Set(9, 0, 1); e != nil {
		t.Fatal(e)
	}
	if v, _ := x.Get(0, 1); v != 9 {
		t.Fatal(v)
	}
}
func TestPriorityQueue(t *testing.T) {
	q := NewPriorityQueue()
	q.Enqueue(strv("low"), 1)
	q.Enqueue(strv("high"), 9)
	v, _ := q.Dequeue()
	if v.Value.Data != UTF8String("high") {
		t.Fatal(v)
	}
}
func TestBlockingQueue(t *testing.T) {
	q := NewBlockingQueue()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() { time.Sleep(10 * time.Millisecond); _ = q.Enqueue(strv("x")) }()
	v, e := q.Dequeue(ctx)
	if e != nil || v.Data != UTF8String("x") {
		t.Fatalf("%v %v", v, e)
	}
}
func TestReliableQueue(t *testing.T) {
	q := NewReliableQueue(2)
	q.Enqueue("1", strv("x"))
	m, ok := q.Claim(time.Second)
	if !ok || m.ID != "1" {
		t.Fatal(m)
	}
	if !q.Nack("1") {
		t.Fatal("nack")
	}
	m, ok = q.Claim(time.Second)
	if !ok {
		t.Fatal("claim 2")
	}
	if !q.Ack(m.ID) {
		t.Fatal("ack")
	}
}
func TestStreamConsumerGroup(t *testing.T) {
	s := NewStream()
	id, e := s.Add(map[string]core.Value{"x": strv("1")}, nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.CreateGroup("g", StreamID{}); e != nil {
		t.Fatal(e)
	}
	r, e := s.ReadGroup("g", "c", 10)
	if e != nil || len(r) != 1 {
		t.Fatalf("%v %v", r, e)
	}
	n, e := s.Ack("g", id)
	if e != nil || n != 1 {
		t.Fatalf("%d %v", n, e)
	}
}
func TestPersistentTopicEventLog(t *testing.T) {
	p := NewPersistentTopic(2)
	p.Publish("", strv("a"))
	p.Publish("", strv("b"))
	p.Publish("", strv("c"))
	if len(p.Read(0, 10)) != 2 {
		t.Fatal("topic retention")
	}
	l := NewEventLog()
	l.Append("created", "1", strv("x"))
	if l.Len() != 1 {
		t.Fatal("event log")
	}
}
func TestBits(t *testing.T) {
	b := NewBitmap()
	b.Set(100, true)
	if !b.Get(100) || b.Count() != 1 {
		t.Fatal("bitmap")
	}
	f := NewBitField(0)
	if e := f.SetUnsigned(3, 5, 17); e != nil {
		t.Fatal(e)
	}
	v, _ := f.GetUnsigned(3, 5)
	if v != 17 {
		t.Fatal(v)
	}
	r := NewRoaringBitmap()
	r.Add(1)
	r.Add(70000)
	if !r.Contains(70000) || r.Cardinality() != 2 {
		t.Fatal(r.Values())
	}
}
func TestBloomAndCounting(t *testing.T) {
	b, e := NewBloomFilter(1000, 0.01)
	if e != nil {
		t.Fatal(e)
	}
	b.Add([]byte("x"))
	if !b.Contains([]byte("x")) {
		t.Fatal("bloom")
	}
	c, _ := NewCountingBloomFilter(1000, 0.01)
	c.Add([]byte("x"))
	if !c.Contains([]byte("x")) || !c.Remove([]byte("x")) || c.Contains([]byte("x")) {
		t.Fatal("counting bloom")
	}
}
func TestCuckooCMSAndTopK(t *testing.T) {
	c := NewCuckooFilter(100)
	if !c.Add([]byte("x")) || !c.Contains([]byte("x")) || !c.Remove([]byte("x")) {
		t.Fatal("cuckoo")
	}
	cms, _ := NewCountMinSketch(128, 4)
	cms.Add([]byte("x"), 5)
	if cms.Estimate([]byte("x")) < 5 {
		t.Fatal("cms")
	}
	top := NewTopK(1)
	top.Add("a", 1)
	top.Add("b", 4)
	if top.Items()[0].Item != "b" {
		t.Fatal(top.Items())
	}
}
func TestHyperLogLog(t *testing.T) {
	h, _ := NewHyperLogLog(10)
	for i := 0; i < 1000; i++ {
		h.Add([]byte(string(rune(i + 1000))))
	}
	n := h.Count()
	if n < 700 || n > 1300 {
		t.Fatalf("estimate=%d", n)
	}
}
func TestTDigest(t *testing.T) {
	d := NewTDigest(50)
	for i := 0; i < 100; i++ {
		d.Add(float64(i), 1)
	}
	q := d.Quantile(.5)
	if q < 35 || q > 65 {
		t.Fatal(q)
	}
}
func TestTimeSeries(t *testing.T) {
	s := NewTimeSeries(map[string]string{"host": "a"}, 0)
	s.Add(1000, 1)
	s.Add(1100, 3)
	r, e := s.Aggregate(1000, 1199, 200, AggAvg)
	if e != nil || len(r) != 1 || r[0].Value != 2 {
		t.Fatalf("%v %v", r, e)
	}
	c := NewCounterSeries(nil, 0)
	c.Add(1, 2)
	c.Add(2, 3)
	p, _ := c.Series.Last()
	if p.Value != 5 {
		t.Fatal(p)
	}
}
func TestHistogram(t *testing.T) {
	h := NewHistogram([]float64{1, 5})
	h.Observe(.5)
	h.Observe(3)
	h.Observe(9)
	_, counts, n, _ := h.Snapshot()
	if n != 3 || counts[0] != 1 || counts[1] != 1 || counts[2] != 1 {
		t.Fatal(counts)
	}
}
func TestGeo(t *testing.T) {
	a, _ := NewGeoPoint(14.5995, 120.9842)
	b, _ := NewGeoPoint(14.6091, 121.0223)
	if d := a.DistanceMeters(b); d < 3000 || d > 6000 {
		t.Fatal(d)
	}
	poly, _ := NewPolygon([]GeoPoint{{0, 0}, {0, 2}, {2, 2}, {2, 0}})
	if !poly.Contains(GeoPoint{1, 1}) {
		t.Fatal("polygon")
	}
	idx := NewGeoSpatialIndex()
	idx.Add("a", GeoPoint{0, 0}, nil)
	if len(idx.WithinRadius(GeoPoint{0, 0}, 100, 10)) != 1 {
		t.Fatal("geo index")
	}
}
func TestVectors(t *testing.T) {
	a := DenseVector{1, 0}
	b := DenseVector{0, 1}
	d, _ := VectorDistance(a, b, MetricCosine)
	if math.Abs(d-1) > 1e-6 {
		t.Fatal(d)
	}
	s, _ := NewVectorSet(2, MetricEuclidean)
	s.Add("a", a, nil)
	s.Add("b", b, nil)
	r, _ := s.Search(DenseVector{.9, .1}, 1, nil)
	if r[0].ID != "a" {
		t.Fatal(r)
	}
	q := QuantizeVector(DenseVector{-.5, 1})
	if len(q.Dequantize()) != 2 {
		t.Fatal("quantize")
	}
}
func TestHNSW(t *testing.T) {
	h, _ := NewHNSWIndex(2, MetricEuclidean, 4, 20)
	_ = h.Add("a", DenseVector{0, 0}, nil)
	_ = h.Add("b", DenseVector{10, 10}, nil)
	r, e := h.Search(DenseVector{.1, .1}, 1, nil)
	if e != nil || len(r) != 1 || r[0].ID != "a" {
		t.Fatalf("%v %v", r, e)
	}
}
func TestIndexes(t *testing.T) {
	b := NewBTreeIndex(2)
	b.Insert("b", "2")
	b.Insert("a", "1")
	if x := b.Search("a"); len(x) != 1 || x[0] != "1" {
		t.Fatal(x)
	}
	if !b.Delete("a", "1") {
		t.Fatal("delete")
	}
	r := NewRadixTree()
	r.Insert("apple", "1")
	r.Insert("app", "2")
	if len(r.Prefix("app", 10)) != 2 {
		t.Fatal("prefix")
	}
	inv := NewInvertedIndex()
	inv.Index("1", "hello world hello")
	if hits := inv.Search("hello", 10); len(hits) != 1 || hits[0].ID != "1" {
		t.Fatal(hits)
	}
}
func TestAdjacencyAndAtomic(t *testing.T) {
	a := NewAdjacencySet()
	a.Add("a", "b", 2)
	if a.Neighbors("a")["b"] != 2 {
		t.Fatal("adj")
	}
	c := NewAtomicCounter(1)
	if c.Add(2) != 3 || !c.CompareAndSwap(3, 5) {
		t.Fatal(c.Load())
	}
}
func TestLeaseSemaphoreRateLimit(t *testing.T) {
	l := NewLease()
	now := time.Now()
	if !l.Acquire("a", "t", time.Second, now) || l.Acquire("b", "u", time.Second, now) {
		t.Fatal("lease")
	}
	s, _ := NewSemaphore(1)
	if !s.TryAcquire() || s.TryAcquire() || !s.Release() {
		t.Fatal("sem")
	}
	b := NewRateLimitBucket(2, 0)
	if !b.Allow(1, now) || !b.Allow(1, now) || b.Allow(1, now) {
		t.Fatal("bucket")
	}
}
func TestNetworkTypes(t *testing.T) {
	c, e := ParseCIDRNetwork("10.0.0.0/8")
	if e != nil || !c.Contains("10.1.2.3") || c.Contains("11.0.0.1") {
		t.Fatal(c, e)
	}
	u, e := ParseURL("https://example.com/a")
	if e != nil || u.Host() != "example.com" {
		t.Fatal(u, e)
	}
}

func TestJSONDocumentPreservesLargeInteger(t *testing.T) {
	j, err := NewJSONDocument([]byte(`{"n":9007199254740993,"u":18446744073709551615}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := j.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `9007199254740993`) || !strings.Contains(string(b), `18446744073709551615`) {
		t.Fatalf("large integers lost precision: %s", b)
	}
}
