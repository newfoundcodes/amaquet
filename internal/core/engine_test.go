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
	"context"
	"errors"
	"testing"
	"time"
)

func TestEngineSetGetDelete(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	v := Value{Type: TypeUTF8String, Data: "hello"}
	if err := e.Set("a", v, 0, false, false); err != nil {
		t.Fatal(err)
	}

	got, ver, err := e.Get("a")
	if err != nil || ver != 1 || got.Type != TypeUTF8String {
		t.Fatalf("got=%v ver=%d err=%v", got, ver, err)
	}

	if n := e.Delete("a"); n != 1 {
		t.Fatalf("deleted %d", n)
	}

	if _, _, err = e.Get("a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found: %v", err)
	}
}

func TestEngineNXAndXX(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	v := Value{Type: TypeInteger, Data: int64(1)}
	if err := e.Set("a", v, 0, false, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("XX: %v", err)
	}

	if err := e.Set("a", v, 0, true, false); err != nil {
		t.Fatal(err)
	}

	if err := e.Set("a", v, 0, true, false); !errors.Is(err, ErrExists) {
		t.Fatalf("NX: %v", err)
	}
}

func TestEngineTTL(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	if err := e.Set("a", Value{Type: TypeNull}, 20*time.Millisecond, false, false); err != nil {
		t.Fatal(err)
	}

	time.Sleep(30 * time.Millisecond)
	if e.Exists("a") {
		t.Fatal("expired key still exists")
	}
}

func TestEngineExpirePersist(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	_ = e.Set("a", Value{Type: TypeNull}, 0, false, false)
	if err := e.Expire("a", time.Second); err != nil {
		t.Fatal(err)
	}

	ttl, _ := e.TTL("a")
	if ttl <= 0 {
		t.Fatalf("ttl=%v", ttl)
	}

	if err := e.Persist("a"); err != nil {
		t.Fatal(err)
	}

	ttl, _ = e.TTL("a")
	if ttl != -1 {
		t.Fatalf("ttl=%v", ttl)
	}
}

func TestEngineCAS(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	_ = e.Set("a", Value{Type: TypeInteger, Data: int64(1)}, 0, false, false)
	if err := e.CompareAndSet("a", 2, Value{Type: TypeInteger, Data: int64(2)}, 0); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict: %v", err)
	}

	if err := e.CompareAndSet("a", 1, Value{Type: TypeInteger, Data: int64(2)}, 0); err != nil {
		t.Fatal(err)
	}

	_, ver, _ := e.Get("a")
	if ver != 2 {
		t.Fatalf("ver=%d", ver)
	}
}

func TestEngineUpdateAndBump(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	_ = e.Set("a", Value{Type: TypeInteger, Data: int64(1)}, 0, false, false)
	_, err := e.Update("a", TypeInteger, func(v Value) (Value, error) { v.Data = int64(2); return v, nil })

	if err != nil {
		t.Fatal(err)
	}

	_, ver, _ := e.Get("a")
	if ver != 2 {
		t.Fatalf("ver=%d", ver)
	}

	if err = e.BumpVersion("a"); err != nil {
		t.Fatal(err)
	}

	_, ver, _ = e.Get("a")
	if ver != 3 {
		t.Fatalf("ver=%d", ver)
	}
}

func TestEngineTransaction(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	_ = e.Set("a", Value{Type: TypeInteger, Data: int64(1)}, 0, false, false)
	_ = e.Set("b", Value{Type: TypeInteger, Data: int64(2)}, 0, false, false)

	if err := e.Transaction(context.Background(), []string{"b", "a"}, func(view map[string]*Entry) error {
		if len(view) != 2 {
			t.Fatalf("view=%d", len(view))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestAllDataTypesStable(t *testing.T) {
	all := AllDataTypes()
	if len(all) < 80 {
		t.Fatalf("expected broad type registry, got %d", len(all))
	}

	seen := map[string]bool{}
	for _, dt := range all {
		name := dt.String()
		if seen[name] {
			t.Fatalf("duplicate %s", name)
		}

		seen[name] = true
		if parsed, err := ParseDataType(name); err != nil || parsed != dt {
			t.Fatalf("round trip %s: %v %v", name, parsed, err)
		}
	}
}
