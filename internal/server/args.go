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
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type args map[string]json.RawMessage

func (a args) string(name string) (string, error) {
	r, ok := a[name]
	if !ok {
		return "", fmt.Errorf("missing %s", name)
	}
	var s string
	if err := json.Unmarshal(r, &s); err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return s, nil
}
func (a args) optString(name, def string) string {
	r, ok := a[name]
	if !ok {
		return def
	}
	var s string
	if json.Unmarshal(r, &s) != nil {
		return def
	}
	return s
}
func (a args) int(name string) (int, error) {
	r, ok := a[name]
	if !ok {
		return 0, fmt.Errorf("missing %s", name)
	}
	var n int
	if err := json.Unmarshal(r, &n); err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return n, nil
}
func (a args) int64(name string) (int64, error) {
	r, ok := a[name]
	if !ok {
		return 0, fmt.Errorf("missing %s", name)
	}
	var n int64
	if err := json.Unmarshal(r, &n); err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return n, nil
}
func (a args) uint64(name string) (uint64, error) {
	r, ok := a[name]
	if !ok {
		return 0, fmt.Errorf("missing %s", name)
	}
	var n uint64
	if err := json.Unmarshal(r, &n); err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return n, nil
}
func (a args) float(name string) (float64, error) {
	r, ok := a[name]
	if !ok {
		return 0, fmt.Errorf("missing %s", name)
	}
	var n float64
	if err := json.Unmarshal(r, &n); err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return n, nil
}
func (a args) bool(name string, def bool) bool {
	r, ok := a[name]
	if !ok {
		return def
	}
	var v bool
	if json.Unmarshal(r, &v) != nil {
		return def
	}
	return v
}
func (a args) duration(name string, def time.Duration) time.Duration {
	r, ok := a[name]
	if !ok {
		return def
	}
	var ms int64
	if json.Unmarshal(r, &ms) == nil {
		return time.Duration(ms) * time.Millisecond
	}
	var s string
	if json.Unmarshal(r, &s) == nil {
		if d, e := time.ParseDuration(s); e == nil {
			return d
		}
	}
	return def
}
func (a args) raw(name string) json.RawMessage { return a[name] }
func (a args) strings(name string) ([]string, error) {
	r, ok := a[name]
	if !ok {
		return nil, fmt.Errorf("missing %s", name)
	}
	var v []string
	if err := json.Unmarshal(r, &v); err != nil {
		return nil, err
	}
	return v, nil
}
func rawToInt(raw json.RawMessage, def int) int {
	if len(raw) == 0 {
		return def
	}
	var n int
	if json.Unmarshal(raw, &n) == nil {
		return n
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if x, e := strconv.Atoi(s); e == nil {
			return x
		}
	}
	return def
}
