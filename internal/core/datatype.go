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

import "fmt"

type DataType uint16

const (
	TypeInvalid DataType = iota
	TypeBinaryString
	TypeUTF8String
	TypeInteger
	TypeUnsignedInteger
	TypeFloat
	TypeDecimal
	TypeBoolean
	TypeNull
	TypeTimestamp
	TypeDuration
	TypeUUID
	TypeBigInteger
	TypeBigDecimal
	TypeSymbol
	TypeList
	TypeArray
	TypeDeque
	TypeRingBuffer
	TypeTuple
	TypeSet
	TypeSortedSet
	TypeOrderedSet
	TypeMultiSet
	TypeHashMap
	TypeOrderedMap
	TypeMultiMap
	TypeJSON
	TypeMessagePack
	TypeCBOR
	TypeRecord
	TypeMatrix
	TypeTensor
	TypeFIFOQueue
	TypeLIFOStack
	TypePriorityQueue
	TypeBlockingQueue
	TypeDelayedQueue
	TypeReliableQueue
	TypeStream
	TypeConsumerGroup
	TypePubSub
	TypePersistentTopic
	TypeEventLog
	TypeBitmap
	TypeBitField
	TypeBitSet
	TypeRoaringBitmap
	TypeHyperLogLog
	TypeBloomFilter
	TypeCountingBloomFilter
	TypeCuckooFilter
	TypeCountMinSketch
	TypeTopK
	TypeTDigest
	TypeTimeSeries
	TypeCounterSeries
	TypeGaugeSeries
	TypeHistogram
	TypeTimeSeriesLabels
	TypeAggregatedSeries
	TypeGeoPoint
	TypeGeoSpatialIndex
	TypeBoundingBox
	TypePolygon
	TypeDenseVector
	TypeSparseVector
	TypeVectorSet
	TypeVectorMetadata
	TypeQuantizedVector
	TypeBTreeIndex
	TypeHashIndex
	TypeRadixTree
	TypeTrie
	TypeInvertedIndex
	TypeHNSWIndex
	TypeFlatVectorIndex
	TypeGeospatialSecondaryIndex
	TypeSecondaryIndex
	TypeFullTextDocument
	TypeTagField
	TypeNumericField
	TypeTextField
	TypeAdjacencySet
	TypeAtomicCounter
	TypeLease
	TypeLock
	TypeBarrier
	TypeSemaphore
	TypeRateLimitBucket
	TypeCIDRNetwork
	TypeURL
)

var typeNames = map[DataType]string{
	TypeBinaryString:             "binary_string",
	TypeUTF8String:               "utf8_string",
	TypeInteger:                  "integer",
	TypeUnsignedInteger:          "unsigned_integer",
	TypeFloat:                    "float",
	TypeDecimal:                  "decimal",
	TypeBoolean:                  "boolean",
	TypeNull:                     "null",
	TypeTimestamp:                "timestamp",
	TypeDuration:                 "duration",
	TypeUUID:                     "uuid",
	TypeBigInteger:               "big_integer",
	TypeBigDecimal:               "big_decimal",
	TypeSymbol:                   "symbol",
	TypeList:                     "list",
	TypeArray:                    "array",
	TypeDeque:                    "deque",
	TypeRingBuffer:               "ring_buffer",
	TypeTuple:                    "tuple",
	TypeSet:                      "set",
	TypeSortedSet:                "sorted_set",
	TypeOrderedSet:               "ordered_set",
	TypeMultiSet:                 "multiset",
	TypeHashMap:                  "hashmap",
	TypeOrderedMap:               "ordered_map",
	TypeMultiMap:                 "multimap",
	TypeJSON:                     "json",
	TypeMessagePack:              "messagepack",
	TypeCBOR:                     "cbor",
	TypeRecord:                   "record",
	TypeMatrix:                   "matrix",
	TypeTensor:                   "tensor",
	TypeFIFOQueue:                "fifo_queue",
	TypeLIFOStack:                "lifo_stack",
	TypePriorityQueue:            "priority_queue",
	TypeBlockingQueue:            "blocking_queue",
	TypeDelayedQueue:             "delayed_queue",
	TypeReliableQueue:            "reliable_queue",
	TypeStream:                   "stream",
	TypeConsumerGroup:            "consumer_group",
	TypePubSub:                   "pubsub",
	TypePersistentTopic:          "persistent_topic",
	TypeEventLog:                 "event_log",
	TypeBitmap:                   "bitmap",
	TypeBitField:                 "bit_field",
	TypeBitSet:                   "bit_set",
	TypeRoaringBitmap:            "roaring_bitmap",
	TypeHyperLogLog:              "hyperloglog",
	TypeBloomFilter:              "bloom_filter",
	TypeCountingBloomFilter:      "counting_bloom_filter",
	TypeCuckooFilter:             "cuckoo_filter",
	TypeCountMinSketch:           "count_min_sketch",
	TypeTopK:                     "top_k",
	TypeTDigest:                  "t_digest",
	TypeTimeSeries:               "timeseries",
	TypeCounterSeries:            "counter_series",
	TypeGaugeSeries:              "gauge_series",
	TypeHistogram:                "histogram",
	TypeTimeSeriesLabels:         "timeseries_labels",
	TypeAggregatedSeries:         "aggregated_series",
	TypeGeoPoint:                 "geo_point",
	TypeGeoSpatialIndex:          "geo_spatial_index",
	TypeBoundingBox:              "bounding_box",
	TypePolygon:                  "polygon",
	TypeDenseVector:              "dense_vector",
	TypeSparseVector:             "sparse_vector",
	TypeVectorSet:                "vector_set",
	TypeVectorMetadata:           "vector_metadata",
	TypeQuantizedVector:          "quantized_vector",
	TypeBTreeIndex:               "btree_index",
	TypeHashIndex:                "hash_index",
	TypeRadixTree:                "radix_tree",
	TypeTrie:                     "trie",
	TypeInvertedIndex:            "inverted_index",
	TypeHNSWIndex:                "hnsw_index",
	TypeFlatVectorIndex:          "flat_vector_index",
	TypeGeospatialSecondaryIndex: "geospatial_secondary_index",
	TypeSecondaryIndex:           "secondary_index",
	TypeFullTextDocument:         "full_text_document",
	TypeTagField:                 "tag_field",
	TypeNumericField:             "numeric_field",
	TypeTextField:                "text_field",
	TypeAdjacencySet:             "adjacency_set",
	TypeAtomicCounter:            "atomic_counter",
	TypeLease:                    "lease",
	TypeLock:                     "lock",
	TypeBarrier:                  "barrier",
	TypeSemaphore:                "semaphore",
	TypeRateLimitBucket:          "rate_limit_bucket",
	TypeCIDRNetwork:              "cidr_network",
	TypeURL:                      "url",
}

func (t DataType) String() string {
	if s, ok := typeNames[t]; ok {
		return s
	}

	return fmt.Sprintf("unknown(%d)", t)
}

func ParseDataType(s string) (DataType, error) {
	for k, v := range typeNames {
		if v == s {
			return k, nil
		}
	}

	return TypeInvalid, fmt.Errorf("unknown data type %q", s)
}

func AllDataTypes() []DataType {
	out := make([]DataType, 0, len(typeNames))
	for i := TypeBinaryString; i <= TypeURL; i++ {
		out = append(out, i)
	}

	return out
}
