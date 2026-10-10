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
	"fmt"
	"testing"
	"time"
)

func BenchmarkSetGet(b *testing.B) {
	e := NewEngine()
	defer e.Close()

	v := Value{Type: TypeUTF8String, Data: "benchmark-value"}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench:%d", i&1023)
		if err := e.Set(key, v, 0, false, false); err != nil {
			b.Fatal(err)
		}

		if _, _, err := e.Get(key); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrentSetGet(b *testing.B) {
	e := NewEngine()
	defer e.Close()

	v := Value{Type: TypeInteger, Data: int64(1)}
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("parallel:%d", i&4095)
			_ = e.Set(key, v, 0, false, false)
			_, _, _ = e.Get(key)
			i++
		}
	})
}

func BenchmarkExpirePersistChurn(b *testing.B) {
	e := NewEngine()
	defer e.Close()

	if err := e.Set("ttl", Value{
		Type: TypeInteger,
		Data: int64(1),
	}, 0, false, false); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := e.Expire("ttl", time.Hour); err != nil {
			b.Fatal(err)
		}

		if err := e.Persist("ttl"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScanPage(b *testing.B) {
	e := NewEngine()
	defer e.Close()

	v := Value{Type: TypeInteger, Data: int64(1)}
	for i := 0; i < 10000; i++ {
		if err := e.Set(fmt.Sprintf("scan:%06d", i), v, 0, false, false); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = e.Scan("", "scan:", 250)
	}
}
