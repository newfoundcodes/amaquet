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
	"errors"
	"testing"
	"time"
)

func FuzzEngineKeyLifecycle(f *testing.F) {
	f.Add("normal:key", int64(1), int64(0))
	f.Add("emoji:🧠", int64(-1), int64(10))
	f.Add("", int64(0), int64(0))

	f.Fuzz(func(t *testing.T, key string, value int64, ttlMs int64) {
		if len(key) > 64<<10 {
			return
		}

		ttlMs %= 1000
		if ttlMs < 0 {
			ttlMs = -ttlMs
		}

		ttl := time.Duration(ttlMs) * time.Millisecond
		expiresNoEarlierThan := time.Now().Add(ttl)

		e := NewEngine()
		defer e.Close()

		err := e.Set(key, Value{Type: TypeInteger, Data: value}, ttl, false, false)
		if key == "" {
			if err == nil {
				t.Fatalf("empty key accepted")
			}
			return
		}

		if err != nil {
			t.Fatalf("set: %v", err)
		}

		got, _, err := e.Get(key)
		if errors.Is(err, ErrNotFound) && ttl > 0 && !time.Now().Before(expiresNoEarlierThan) {
			if e.Exists(key) {
				t.Fatalf("expired key still exists")
			}

			return
		}

		if err != nil {
			t.Fatalf("get: %v", err)
		}

		if got.Type != TypeInteger {
			t.Fatalf("type changed")
		}

		if n := e.Delete(key); n != 1 {
			if ttl == 0 || n != 0 || time.Now().Before(expiresNoEarlierThan) {
				t.Fatalf("delete count=%d", n)
			}
		}

		if e.Exists(key) {
			t.Fatalf("key still exists")
		}
	})
}
