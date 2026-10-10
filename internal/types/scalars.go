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
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"
)

type BinaryString []byte

type UTF8String string

func NewUTF8String(s string) (UTF8String, error) {
	if !utf8.ValidString(s) {
		return "", fmt.Errorf("invalid UTF-8")
	}
	return UTF8String(s), nil
}

type Integer int64
type UnsignedInteger uint64
type Float float64
type Boolean bool
type Null struct{}
type Timestamp time.Time
type Duration time.Duration

type Decimal struct {
	Coeff big.Int `json:"-"`
	Scale int32   `json:"scale"`
}

func ParseDecimal(s string) (Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Decimal{}, fmt.Errorf("empty decimal")
	}
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return Decimal{}, fmt.Errorf("invalid decimal")
	}
	whole := parts[0]
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if whole == "" {
		whole = "0"
	}
	digits := whole + frac
	for _, r := range digits {
		if r < '0' || r > '9' {
			return Decimal{}, fmt.Errorf("invalid decimal")
		}
	}
	var z big.Int
	if _, ok := z.SetString(digits, 10); !ok {
		return Decimal{}, fmt.Errorf("invalid decimal")
	}
	if neg {
		z.Neg(&z)
	}
	return Decimal{Coeff: z, Scale: int32(len(frac))}, nil
}
func (d Decimal) String() string {
	neg := d.Coeff.Sign() < 0
	var a big.Int
	a.Abs(&d.Coeff)
	s := a.String()
	scale := int(d.Scale)
	if scale > 0 {
		if len(s) <= scale {
			s = strings.Repeat("0", scale-len(s)+1) + s
		}
		s = s[:len(s)-scale] + "." + s[len(s)-scale:]
	}
	if neg {
		return "-" + s
	}
	return s
}
func (d Decimal) Add(o Decimal) Decimal {
	a, b, scale := alignDecimals(d, o)
	var z big.Int
	z.Add(&a, &b)
	return Decimal{Coeff: z, Scale: scale}
}
func (d Decimal) Sub(o Decimal) Decimal {
	a, b, scale := alignDecimals(d, o)
	var z big.Int
	z.Sub(&a, &b)
	return Decimal{Coeff: z, Scale: scale}
}
func (d Decimal) Cmp(o Decimal) int { a, b, _ := alignDecimals(d, o); return a.Cmp(&b) }
func alignDecimals(a, b Decimal) (big.Int, big.Int, int32) {
	scale := a.Scale
	if b.Scale > scale {
		scale = b.Scale
	}
	var x, y big.Int
	x.Set(&a.Coeff)
	y.Set(&b.Coeff)
	ten := big.NewInt(10)
	if a.Scale < scale {
		var p big.Int
		p.Exp(ten, big.NewInt(int64(scale-a.Scale)), nil)
		x.Mul(&x, &p)
	}
	if b.Scale < scale {
		var p big.Int
		p.Exp(ten, big.NewInt(int64(scale-b.Scale)), nil)
		y.Mul(&y, &p)
	}
	return x, y, scale
}

type BigInteger struct {
	Int big.Int `json:"-"`
}

func ParseBigInteger(s string) (BigInteger, error) {
	var z big.Int
	if _, ok := z.SetString(strings.TrimSpace(s), 10); !ok {
		return BigInteger{}, fmt.Errorf("invalid big integer")
	}
	return BigInteger{Int: z}, nil
}
func (b BigInteger) String() string { return b.Int.String() }

type BigDecimal = Decimal

type UUID [16]byte

func ParseUUID(s string) (UUID, error) {
	var u UUID
	s = strings.ReplaceAll(strings.TrimSpace(s), "-", "")
	if len(s) != 32 {
		return u, fmt.Errorf("invalid UUID")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return u, err
	}
	copy(u[:], b)
	return u, nil
}
func (u UUID) String() string {
	h := hex.EncodeToString(u[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

type Symbol string

func NewSymbol(s string) (Symbol, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, " \t\r\n") {
		return "", fmt.Errorf("symbol must be non-empty and contain no whitespace")
	}
	return Symbol(s), nil
}
