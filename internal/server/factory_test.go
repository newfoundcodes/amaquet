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

func raw(v any) json.RawMessage { b, _ := json.Marshal(v); return b }

func TestEveryRegisteredTypeHasAConstructionPath(t *testing.T) {
	direct := map[core.DataType]json.RawMessage{
		core.TypeBinaryString:    raw("AAEC"),
		core.TypeUTF8String:      raw("hello"),
		core.TypeInteger:         raw(int64(-1)),
		core.TypeUnsignedInteger: raw(uint64(1)),
		core.TypeFloat:           raw(1.5),
		core.TypeDecimal:         raw("12.34"),
		core.TypeBoolean:         raw(true),
		core.TypeNull:            raw(nil),
		core.TypeTimestamp:       raw("2026-10-01T00:00:00Z"),
		core.TypeDuration:        raw("5s"),
		core.TypeUUID:            raw("550e8400-e29b-41d4-a716-446655440000"),
		core.TypeBigInteger:      raw("123456789012345678901234567890"),
		core.TypeBigDecimal:      raw("1234567890.0000000001"),
		core.TypeSymbol:          raw("ACTIVE"),
		core.TypeGeoPoint:        raw(map[string]any{"Lat": 14.5, "Lon": 121.0}),
		core.TypeBoundingBox:     raw(map[string]any{"MinLat": 14.0, "MinLon": 120.0, "MaxLat": 15.0, "MaxLon": 122.0}),
		core.TypePolygon:         raw([]map[string]any{{"Lat": 14.0, "Lon": 120.0}, {"Lat": 15.0, "Lon": 120.0}, {"Lat": 14.5, "Lon": 121.0}}),
		core.TypeDenseVector:     raw([]float32{1, 2, 3}),
		core.TypeSparseVector:    raw(map[string]any{"dim": 3, "values": map[int]float32{1: 2}}),
		core.TypeQuantizedVector: raw(map[string]any{"Data": []int8{1, 2, 3}, "Scale": 0.1, "ZeroPoint": 0}),
		core.TypeCIDRNetwork:     raw("10.0.0.0/8"),
		core.TypeURL:             raw("https://example.com/path"),
	}
	for _, typ := range core.AllDataTypes() {
		if r, ok := direct[typ]; ok {
			v, err := decodeTypedValue(typ, r)
			if err != nil {
				t.Errorf("SET construction for %s failed: %v", typ, err)
				continue
			}
			if v.Type != typ {
				t.Errorf("SET construction returned %s for %s", v.Type, typ)
			}
			continue
		}
		options := args{}
		switch typ {
		case core.TypeVectorSet, core.TypeHNSWIndex, core.TypeFlatVectorIndex:
			options = args{"dim": raw(3)}
		}
		v, err := createValue(typ, options)
		if err != nil {
			t.Errorf("CREATE construction for %s failed: %v", typ, err)
			continue
		}
		if v.Type != typ {
			t.Errorf("CREATE construction returned %s for %s", v.Type, typ)
		}
	}
}
