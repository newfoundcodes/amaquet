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

func TestSameSizeReplacementAtMemoryCeiling(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	v := Value{Type: TypeUTF8String, Data: string(make([]byte, 256))}
	if err := e.Set("k", v, 0, false, false); err != nil {
		t.Fatal(err)
	}

	used := e.MemoryUsage()
	if err := e.ConfigureMemory(used, "noeviction"); err != nil {
		t.Fatal(err)
	}

	if err := e.Set("k", v, 0, false, false); err != nil {
		t.Fatalf("same-size replacement failed at ceiling: %v", err)
	}

	larger := Value{Type: TypeUTF8String, Data: string(make([]byte, 512))}
	if err := e.Set("k", larger, 0, false, false); !errors.Is(err, ErrMemoryLimit) {
		t.Fatalf("larger replacement error=%v", err)
	}
}

func TestExpiryHeapMaintainsSingleNodePerKey(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	if err := e.Set("k", Value{Type: TypeInteger, Data: int64(1)}, 0, false, false); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 1000; i++ {
		if err := e.Expire("k", time.Hour+time.Duration(i)*time.Millisecond); err != nil {
			t.Fatal(err)
		}
	}
	e.expMu.Lock()

	n := len(e.expiries)
	indexed := len(e.expiryByKey)
	e.expMu.Unlock()

	if n != 1 || indexed != 1 {
		t.Fatalf("expiry heap leaked stale nodes: heap=%d index=%d", n, indexed)
	}

	if err := e.Persist("k"); err != nil {
		t.Fatal(err)
	}
	e.expMu.Lock()

	n = len(e.expiries)
	indexed = len(e.expiryByKey)
	e.expMu.Unlock()

	if n != 0 || indexed != 0 {
		t.Fatalf("persist did not unschedule expiry: heap=%d index=%d", n, indexed)
	}
}

func TestTransactionRollbackDoesNotTouchLiveEntry(t *testing.T) {
	e := NewEngine()
	defer e.Close()

	if err := e.Set("k", Value{Type: TypeInteger, Data: int64(1)}, 0, false, false); err != nil {
		t.Fatal(err)
	}

	wantErr := errors.New("rollback")
	err := e.Transaction(context.Background(), []string{"k"}, func(view map[string]*Entry) error {
		view["k"].Value = Value{Type: TypeInteger, Data: int64(2)}
		return wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("transaction error=%v", err)
	}

	got, _, err := e.Get("k")
	if err != nil {
		t.Fatal(err)
	}

	if got.Data.(int64) != 1 {
		t.Fatalf("rollback leaked value: %v", got.Data)
	}
}
