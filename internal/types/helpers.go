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

package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/newfoundcodes/amaquet/internal/core"
)

func valueKey(v core.Value) string {
	b, err := json.Marshal(v.Data)
	if err != nil {
		return fmt.Sprintf("%d:%v", v.Type, v.Data)
	}
	h := sha256.Sum256(append([]byte{byte(v.Type >> 8), byte(v.Type)}, b...))
	return hex.EncodeToString(h[:])
}

func cloneValues(in []core.Value) []core.Value {
	out := make([]core.Value, len(in))
	copy(out, in)
	return out
}
