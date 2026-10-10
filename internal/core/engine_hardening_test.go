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
	"testing"
	"time"
)

func TestExpiredKeyCannotBeRevived(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	if err := e.Set("k", Value{Type: TypeUTF8String, Data: "v"}, 15*time.Millisecond, false, false); err != nil {
		t.Fatal(err)
	}

	time.Sleep(30 * time.Millisecond)
	if err := e.Persist("k"); err != ErrNotFound {
		t.Fatalf("Persist expired: got %v want %v", err, ErrNotFound)
	}

	if err := e.Expire("k", time.Second); err != ErrNotFound {
		t.Fatalf("Expire expired: got %v want %v", err, ErrNotFound)
	}

	if e.Exists("k") {
		t.Fatal("expired key revived")
	}
}

func TestLenAndScan(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	for _, k := range []string{"user:3", "user:1", "other", "user:2"} {
		if err := e.Set(k, Value{Type: TypeUTF8String, Data: k}, 0, false, false); err != nil {
			t.Fatal(err)
		}
	}

	if got := e.Len(); got != 4 {
		t.Fatalf("Len=%d", got)
	}

	p1, c := e.Scan("", "user:", 2)
	if len(p1) != 2 || c == "" {
		t.Fatalf("p1=%v c=%q", p1, c)
	}

	p2, c2 := e.Scan(c, "user:", 2)
	if len(p2) != 1 || c2 != "" {
		t.Fatalf("p2=%v c2=%q", p2, c2)
	}
}

func TestMemoryLimitNoEvictionStillAllowsDelete(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	if err := e.ConfigureMemory(320, "noeviction"); err != nil {
		t.Fatal(err)
	}

	if err := e.Set("a", Value{Type: TypeUTF8String, Data: string(make([]byte, 120))}, 0, false, false); err != nil {
		t.Fatal(err)
	}

	err := e.Set("b", Value{Type: TypeUTF8String, Data: string(make([]byte, 120))}, 0, false, false)
	if err != ErrMemoryLimit {
		t.Fatalf("second set error=%v", err)
	}

	if n := e.Delete("a"); n != 1 {
		t.Fatalf("delete=%d", n)
	}

	if e.MemoryUsage() != 0 {
		t.Fatalf("memory not released: %d", e.MemoryUsage())
	}
}

func TestMemoryEviction(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	if err := e.ConfigureMemory(430, "allkeys-lru"); err != nil {
		t.Fatal(err)
	}

	for _, k := range []string{"a", "b", "c"} {
		if err := e.Set(k, Value{Type: TypeUTF8String, Data: string(make([]byte, 120))}, 0, false, false); err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Millisecond)
	}

	if e.Len() >= 3 {
		t.Fatalf("expected eviction, keys=%d stats=%+v", e.Len(), e.MemoryStats())
	}

	if e.MemoryStats().Evicted == 0 {
		t.Fatal("eviction counter did not advance")
	}
}
