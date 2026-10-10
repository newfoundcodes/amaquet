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
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"
)

type Value struct {
	Type DataType `json:"type"`
	Data any      `json:"value"`
}

type Entry struct {
	Value      Value     `json:"value"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
	Version    uint64    `json:"version"`
	accesses   atomic.Uint64
	lastAccess atomic.Int64
	sizeBytes  int64
}

type JSONValue struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

var ErrNotFound = errors.New("key not found")
var ErrWrongType = errors.New("wrong value type")
var ErrExists = errors.New("key already exists")
var ErrInvalidArgument = errors.New("invalid argument")
var ErrConflict = errors.New("version conflict")
var ErrMemoryLimit = errors.New("memory limit reached")

func (e *Entry) Expired(now time.Time) bool {
	return !e.ExpiresAt.IsZero() && !now.Before(e.ExpiresAt)
}

func (e *Entry) Touch() {
	e.accesses.Add(1)
	e.lastAccess.Store(time.Now().UnixNano())
}

func (e *Entry) AccessCount() uint64 {
	return e.accesses.Load()
}

func (e *Entry) LastAccess() time.Time {
	n := e.lastAccess.Load()
	if n == 0 {
		return e.UpdatedAt
	}

	return time.Unix(0, n)
}

func (e *Entry) ApproxBytes() int64 {
	return e.sizeBytes
}
