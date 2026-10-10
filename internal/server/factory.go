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

package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/newfoundcodes/amaquet/internal/core"
	nt "github.com/newfoundcodes/amaquet/internal/types"
)

type wireValue struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

func decodeWireValue(raw json.RawMessage) (core.Value, error) {
	var w wireValue
	if err := json.Unmarshal(raw, &w); err != nil {
		return core.Value{}, err
	}
	t, err := core.ParseDataType(w.Type)
	if err != nil {
		return core.Value{}, err
	}
	return decodeTypedValue(t, w.Value)
}
func decodeTypedValue(t core.DataType, raw json.RawMessage) (core.Value, error) {
	var data any
	switch t {
	case core.TypeBinaryString:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return core.Value{}, err
		}
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return core.Value{}, fmt.Errorf("binary_string expects base64: %w", err)
		}
		data = nt.BinaryString(b)
	case core.TypeUTF8String:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return core.Value{}, err
		}
		v, err := nt.NewUTF8String(s)
		if err != nil {
			return core.Value{}, err
		}
		data = v
	case core.TypeInteger:
		var v int64
		if err := json.Unmarshal(raw, &v); err != nil {
			return core.Value{}, err
		}
		data = nt.Integer(v)
	case core.TypeUnsignedInteger:
		var v uint64
		if err := json.Unmarshal(raw, &v); err != nil {
			return core.Value{}, err
		}
		data = nt.UnsignedInteger(v)
	case core.TypeFloat:
		var v float64
		if err := json.Unmarshal(raw, &v); err != nil {
			return core.Value{}, err
		}
		data = nt.Float(v)
	case core.TypeDecimal, core.TypeBigDecimal:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return core.Value{}, fmt.Errorf("decimal must be a string")
		}
		v, err := nt.ParseDecimal(s)
		if err != nil {
			return core.Value{}, err
		}
		data = v
	case core.TypeBoolean:
		var v bool
		if err := json.Unmarshal(raw, &v); err != nil {
			return core.Value{}, err
		}
		data = nt.Boolean(v)
	case core.TypeNull:
		data = nt.Null{}
	case core.TypeTimestamp:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return core.Value{}, err
		}
		v, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return core.Value{}, err
		}
		data = nt.Timestamp(v)
	case core.TypeDuration:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return core.Value{}, err
		}
		v, err := time.ParseDuration(s)
		if err != nil {
			return core.Value{}, err
		}
		data = nt.Duration(v)
	case core.TypeUUID:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return core.Value{}, err
		}
		v, err := nt.ParseUUID(s)
		if err != nil {
			return core.Value{}, err
		}
		data = v
	case core.TypeBigInteger:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return core.Value{}, err
		}
		v, err := nt.ParseBigInteger(s)
		if err != nil {
			return core.Value{}, err
		}
		data = v
	case core.TypeSymbol:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return core.Value{}, err
		}
		v, err := nt.NewSymbol(s)
		if err != nil {
			return core.Value{}, err
		}
		data = v
	case core.TypeJSON:
		v, err := nt.NewJSONDocument(raw)
		if err != nil {
			return core.Value{}, err
		}
		data = v
	case core.TypeMessagePack:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return core.Value{}, err
		}
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return core.Value{}, err
		}
		data = nt.NewMessagePack(b)
	case core.TypeCBOR:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return core.Value{}, err
		}
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return core.Value{}, err
		}
		data = nt.NewCBOR(b)
	case core.TypeGeoPoint:
		var v struct{ Lat, Lon float64 }
		if err := json.Unmarshal(raw, &v); err != nil {
			return core.Value{}, err
		}
		p, err := nt.NewGeoPoint(v.Lat, v.Lon)
		if err != nil {
			return core.Value{}, err
		}
		data = p
	case core.TypeBoundingBox:
		var v struct{ MinLat, MinLon, MaxLat, MaxLon float64 }
		if err := json.Unmarshal(raw, &v); err != nil {
			return core.Value{}, err
		}
		b, err := nt.NewBoundingBox(v.MinLat, v.MinLon, v.MaxLat, v.MaxLon)
		if err != nil {
			return core.Value{}, err
		}
		data = b
	case core.TypePolygon:
		var pts []nt.GeoPoint
		if err := json.Unmarshal(raw, &pts); err != nil {
			return core.Value{}, err
		}
		p, err := nt.NewPolygon(pts)
		if err != nil {
			return core.Value{}, err
		}
		data = p
	case core.TypeDenseVector:
		var v []float32
		if err := json.Unmarshal(raw, &v); err != nil {
			return core.Value{}, err
		}
		data = nt.DenseVector(v)
	case core.TypeSparseVector:
		var v struct {
			Dim    int             `json:"dim"`
			Values map[int]float32 `json:"values"`
		}
		if err := json.Unmarshal(raw, &v); err != nil {
			return core.Value{}, err
		}
		sv, err := nt.NewSparseVector(v.Dim, v.Values)
		if err != nil {
			return core.Value{}, err
		}
		data = sv
	case core.TypeQuantizedVector:
		var v nt.QuantizedVector
		if err := json.Unmarshal(raw, &v); err != nil {
			return core.Value{}, err
		}
		data = v
	case core.TypeCIDRNetwork:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return core.Value{}, err
		}
		v, err := nt.ParseCIDRNetwork(s)
		if err != nil {
			return core.Value{}, err
		}
		data = v
	case core.TypeURL:
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return core.Value{}, err
		}
		v, err := nt.ParseURL(s)
		if err != nil {
			return core.Value{}, err
		}
		data = v
	default:
		return core.Value{}, fmt.Errorf("type %s must be created with CREATE", t)
	}
	return core.Value{Type: t, Data: data}, nil
}

func createValue(t core.DataType, opt args) (core.Value, error) {
	var d any
	switch t {
	case core.TypeList:
		d = nt.NewList()
	case core.TypeArray:
		et := core.TypeInvalid
		if s := opt.optString("element_type", ""); s != "" {
			var e error
			et, e = core.ParseDataType(s)
			if e != nil {
				return core.Value{}, e
			}
		}
		a, e := nt.NewArray(et, opt.bool("typed", false))
		if e != nil {
			return core.Value{}, e
		}
		d = a
	case core.TypeDeque:
		d = nt.NewDeque(rawToInt(opt.raw("capacity"), 16))
	case core.TypeRingBuffer:
		d = nt.NewRingBuffer(rawToInt(opt.raw("capacity"), 1024))
	case core.TypeTuple:
		vals, e := decodeWireValueArray(opt.raw("values"))
		if e != nil {
			return core.Value{}, e
		}
		d = nt.NewTuple(vals...)
	case core.TypeSet:
		d = nt.NewSet()
	case core.TypeSortedSet:
		d = nt.NewSortedSet()
	case core.TypeOrderedSet:
		d = nt.NewOrderedSet()
	case core.TypeMultiSet:
		d = nt.NewMultiSet()
	case core.TypeHashMap:
		d = nt.NewHashMap()
	case core.TypeOrderedMap:
		d = nt.NewOrderedMap()
	case core.TypeMultiMap:
		d = nt.NewMultiMap()
	case core.TypeJSON:
		v, e := nt.NewJSONDocument([]byte(`{}`))
		if e != nil {
			return core.Value{}, e
		}
		d = v
	case core.TypeMessagePack:
		d = nt.NewMessagePack(nil)
	case core.TypeCBOR:
		d = nt.NewCBOR(nil)
	case core.TypeRecord:
		var schema []struct {
			Name     string `json:"name"`
			Type     string `json:"type"`
			Nullable bool   `json:"nullable"`
		}
		if r := opt.raw("schema"); len(r) > 0 {
			if e := json.Unmarshal(r, &schema); e != nil {
				return core.Value{}, e
			}
		}
		fs := make([]nt.RecordField, 0, len(schema))
		for _, f := range schema {
			dt, e := core.ParseDataType(f.Type)
			if e != nil {
				return core.Value{}, e
			}
			fs = append(fs, nt.RecordField{Name: f.Name, Type: dt, Nullable: f.Nullable})
		}
		d = nt.NewRecord(fs)
	case core.TypeMatrix:
		rows, cols := rawToInt(opt.raw("rows"), 1), rawToInt(opt.raw("cols"), 1)
		data := make([]float64, rows*cols)
		if r := opt.raw("data"); len(r) > 0 {
			if e := json.Unmarshal(r, &data); e != nil {
				return core.Value{}, e
			}
		}
		m, e := nt.NewMatrix(rows, cols, nt.NumericDType(opt.optString("dtype", "float64")), data)
		if e != nil {
			return core.Value{}, e
		}
		d = m
	case core.TypeTensor:
		var shape []int
		if e := json.Unmarshal(opt.raw("shape"), &shape); e != nil || len(shape) == 0 {
			shape = []int{1}
		}
		n := 1
		for _, x := range shape {
			n *= x
		}
		data := make([]float64, n)
		if r := opt.raw("data"); len(r) > 0 {
			if e := json.Unmarshal(r, &data); e != nil {
				return core.Value{}, e
			}
		}
		v, e := nt.NewTensor(shape, nt.NumericDType(opt.optString("dtype", "float64")), data)
		if e != nil {
			return core.Value{}, e
		}
		d = v
	case core.TypeFIFOQueue:
		d = nt.NewFIFOQueue()
	case core.TypeLIFOStack:
		d = nt.NewLIFOStack()
	case core.TypePriorityQueue:
		d = nt.NewPriorityQueue()
	case core.TypeBlockingQueue:
		d = nt.NewBlockingQueue()
	case core.TypeDelayedQueue:
		d = nt.NewDelayedQueue()
	case core.TypeReliableQueue:
		d = nt.NewReliableQueue(rawToInt(opt.raw("max_attempts"), 5))
	case core.TypeStream:
		d = nt.NewStream()
	case core.TypeConsumerGroup:
		d = nt.NewConsumerGroup(opt.optString("name", "default"), nt.StreamID{})
	case core.TypePubSub:
		d = nt.NewPubSub()
	case core.TypePersistentTopic:
		d = nt.NewPersistentTopic(rawToInt(opt.raw("max_messages"), 0))
	case core.TypeEventLog:
		d = nt.NewEventLog()
	case core.TypeBitmap:
		d = nt.NewBitmap()
	case core.TypeBitField:
		d = nt.NewBitField(rawToInt(opt.raw("size_bytes"), 0))
	case core.TypeBitSet:
		d = nt.NewBitmap()
	case core.TypeRoaringBitmap:
		d = nt.NewRoaringBitmap()
	case core.TypeHyperLogLog:
		v, e := nt.NewHyperLogLog(uint8(rawToInt(opt.raw("precision"), 14)))
		if e != nil {
			return core.Value{}, e
		}
		d = v
	case core.TypeBloomFilter:
		v, e := nt.NewBloomFilter(uint64(rawToInt(opt.raw("expected"), 100000)), optFloat(opt, "false_positive_rate", 0.01))
		if e != nil {
			return core.Value{}, e
		}
		d = v
	case core.TypeCountingBloomFilter:
		v, e := nt.NewCountingBloomFilter(uint64(rawToInt(opt.raw("expected"), 100000)), optFloat(opt, "false_positive_rate", 0.01))
		if e != nil {
			return core.Value{}, e
		}
		d = v
	case core.TypeCuckooFilter:
		d = nt.NewCuckooFilter(uint64(rawToInt(opt.raw("capacity"), 100000)))
	case core.TypeCountMinSketch:
		v, e := nt.NewCountMinSketch(uint32(rawToInt(opt.raw("width"), 2048)), uint32(rawToInt(opt.raw("depth"), 5)))
		if e != nil {
			return core.Value{}, e
		}
		d = v
	case core.TypeTopK:
		d = nt.NewTopK(rawToInt(opt.raw("k"), 10))
	case core.TypeTDigest:
		d = nt.NewTDigest(optFloat(opt, "compression", 100))
	case core.TypeTimeSeries:
		d = nt.NewTimeSeries(optLabels(opt), opt.duration("retention_ms", 0))
	case core.TypeCounterSeries:
		d = nt.NewCounterSeries(optLabels(opt), opt.duration("retention_ms", 0))
	case core.TypeGaugeSeries:
		d = nt.NewGaugeSeries(optLabels(opt), opt.duration("retention_ms", 0))
	case core.TypeHistogram:
		var bounds []float64
		_ = json.Unmarshal(opt.raw("bounds"), &bounds)
		d = nt.NewHistogram(bounds)
	case core.TypeTimeSeriesLabels:
		d = nt.TimeSeriesLabels(optLabels(opt))
	case core.TypeAggregatedSeries:
		d = &nt.AggregatedSeries{BucketMillis: int64(rawToInt(opt.raw("bucket_ms"), 60000)), Aggregation: nt.Aggregation(opt.optString("aggregation", "avg"))}
	case core.TypeGeoSpatialIndex, core.TypeGeospatialSecondaryIndex:
		d = nt.NewGeoSpatialIndex()
	case core.TypeVectorSet:
		dim := rawToInt(opt.raw("dim"), 0)
		v, e := nt.NewVectorSet(dim, nt.VectorMetric(opt.optString("metric", "cosine")))
		if e != nil {
			return core.Value{}, e
		}
		d = v
	case core.TypeVectorMetadata:
		d = nt.VectorMetadata{}
	case core.TypeBTreeIndex:
		d = nt.NewBTreeIndex(rawToInt(opt.raw("degree"), 16))
	case core.TypeHashIndex:
		d = nt.NewHashIndex()
	case core.TypeRadixTree:
		d = nt.NewRadixTree()
	case core.TypeTrie:
		d = nt.NewTrie()
	case core.TypeInvertedIndex:
		d = nt.NewInvertedIndex()
	case core.TypeHNSWIndex:
		dim := rawToInt(opt.raw("dim"), 0)
		v, e := nt.NewHNSWIndex(dim, nt.VectorMetric(opt.optString("metric", "cosine")), rawToInt(opt.raw("m"), 16), rawToInt(opt.raw("ef_search"), 64))
		if e != nil {
			return core.Value{}, e
		}
		d = v
	case core.TypeFlatVectorIndex:
		dim := rawToInt(opt.raw("dim"), 0)
		v, e := nt.NewFlatVectorIndex(dim, nt.VectorMetric(opt.optString("metric", "cosine")))
		if e != nil {
			return core.Value{}, e
		}
		d = v
	case core.TypeSecondaryIndex:
		d = nt.NewSecondaryIndex()
	case core.TypeFullTextDocument:
		var meta map[string]any
		_ = json.Unmarshal(opt.raw("metadata"), &meta)
		d = &nt.FullTextDocument{ID: opt.optString("id", ""), Metadata: meta}
	case core.TypeTagField:
		var values []string
		_ = json.Unmarshal(opt.raw("values"), &values)
		d = &nt.TagField{Name: opt.optString("name", ""), Values: values}
	case core.TypeNumericField:
		d = &nt.NumericField{Name: opt.optString("name", ""), Value: optFloat(opt, "value", 0)}
	case core.TypeTextField:
		d = &nt.TextField{Name: opt.optString("name", ""), Value: opt.optString("value", ""), Weight: optFloat(opt, "weight", 1)}
	case core.TypeAdjacencySet:
		d = nt.NewAdjacencySet()
	case core.TypeAtomicCounter:
		d = nt.NewAtomicCounter(int64(rawToInt(opt.raw("initial"), 0)))
	case core.TypeLease:
		d = nt.NewLease()
	case core.TypeLock:
		d = nt.NewLock()
	case core.TypeBarrier:
		v, e := nt.NewBarrier(rawToInt(opt.raw("target"), 1))
		if e != nil {
			return core.Value{}, e
		}
		d = v
	case core.TypeSemaphore:
		v, e := nt.NewSemaphore(rawToInt(opt.raw("limit"), 1))
		if e != nil {
			return core.Value{}, e
		}
		d = v
	case core.TypeRateLimitBucket:
		d = nt.NewRateLimitBucket(optFloat(opt, "capacity", 100), optFloat(opt, "refill_per_second", 10))
	default:
		return core.Value{}, fmt.Errorf("CREATE not required/supported for %s; use SET", t)
	}
	return core.Value{Type: t, Data: d}, nil
}
func decodeWireValueArray(raw json.RawMessage) ([]core.Value, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var rs []json.RawMessage
	if err := json.Unmarshal(raw, &rs); err != nil {
		return nil, err
	}
	out := make([]core.Value, 0, len(rs))
	for _, r := range rs {
		v, err := decodeWireValue(r)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func optFloat(a args, name string, def float64) float64 {
	r := a.raw(name)
	if len(r) == 0 {
		return def
	}
	var v float64
	if json.Unmarshal(r, &v) == nil {
		return v
	}
	return def
}
func optLabels(a args) map[string]string {
	var m map[string]string
	if json.Unmarshal(a.raw("labels"), &m) != nil || m == nil {
		return map[string]string{}
	}
	return m
}

var _ = big.NewInt
