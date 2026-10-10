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

package protocol

import (
	"bytes"
	"reflect"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	in := Frame{Kind: KindRequest, Flags: 7, Opcode: OpCommand, RequestID: 42, Payload: []byte(`{"command":"PING"}`)}
	var b bytes.Buffer
	if err := WriteFrame(&b, in); err != nil {
		t.Fatal(err)
	}
	out, err := ReadFrame(&b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round trip mismatch: %#v != %#v", in, out)
	}
}

func TestParseEndpoint(t *testing.T) {
	cases := []struct {
		raw  string
		host string
		port int
		tls  bool
		db   string
		key  string
	}{
		{"amaquet://127.0.0.1:13378", "127.0.0.1", 13378, false, "", ""},
		{"amaquet://example.com", "example.com", 13378, false, "", ""},
		{"amaquets://example.com:14443?api_key=amaquet_test", "example.com", 14443, true, "", "amaquet_test"},
		{"amaquet://amaquet_secret@example.com", "example.com", 13378, false, "", "amaquet_secret"},
	}
	for _, tc := range cases {
		ep, err := ParseEndpoint(tc.raw)
		if err != nil {
			t.Fatalf("%s: %v", tc.raw, err)
		}
		if ep.Host != tc.host || ep.Port != tc.port || ep.TLS != tc.tls || ep.Database != tc.db || ep.APIKey != tc.key {
			t.Fatalf("%s: unexpected endpoint: %#v", tc.raw, ep)
		}
	}
	for _, bad := range []string{"redis://localhost", "amaquet://", "amaquet://localhost:99999", "amaquet://localhost/app"} {
		if _, err := ParseEndpoint(bad); err == nil {
			t.Fatalf("expected error for %s", bad)
		}
	}
}
