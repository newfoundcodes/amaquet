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
	"testing"

	"github.com/newfoundcodes/amaquet/internal/core"
)

func FuzzDecodeTypedValue(f *testing.F) {
	seeds := []struct{ typ, raw string }{
		{"integer", "1"}, {"utf8_string", `"hello"`}, {"binary_string", `"aGVsbG8="`},
		{"json", `{"a":1}`}, {"uuid", `"550e8400-e29b-41d4-a716-446655440000"`},
		{"dense_vector", `[1,2,3]`}, {"cidr_network", `"10.0.0.0/8"`}, {"url", `"https://example.com"`},
	}
	for _, s := range seeds {
		f.Add(s.typ, []byte(s.raw))
	}
	f.Fuzz(func(t *testing.T, typeName string, raw []byte) {
		if len(raw) > 1<<20 {
			return
		}
		typ, err := core.ParseDataType(typeName)
		if err != nil {
			return
		}
		_, _ = decodeTypedValue(typ, json.RawMessage(raw))
	})
}

func FuzzDecodeWireValue(f *testing.F) {
	f.Add([]byte(`{"type":"integer","value":1}`))
	f.Add([]byte(`{"type":"utf8_string","value":"x"}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 1<<20 {
			return
		}
		_, _ = decodeWireValue(json.RawMessage(raw))
	})
}
