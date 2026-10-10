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
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/newfoundcodes/amaquet/internal/compression"
	"github.com/newfoundcodes/amaquet/internal/core"
	"github.com/newfoundcodes/amaquet/internal/protocol"
)

func request(command string, args map[string]any) protocol.Request {
	r := protocol.Request{Command: command, Args: map[string]json.RawMessage{}}
	for k, v := range args {
		b, _ := json.Marshal(v)
		r.Args[k] = b
	}
	return r
}

func TestTransparentStringCompression(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	d := NewDispatcher(e)
	if err := d.SetCompression(compression.Config{Enabled: true, Algorithm: "auto", MinBytes: 64, MinSavingsPercent: 5}); err != nil {
		t.Fatal(err)
	}
	text := strings.Repeat("amaquet-fast-compression-", 1000)
	if _, err := d.Execute(context.Background(), request("SET", map[string]any{"key": "large", "type": "utf8_string", "value": text})); err != nil {
		t.Fatal(err)
	}
	ent, err := e.GetEntry("large")
	if err != nil {
		t.Fatal(err)
	}
	p, ok := ent.Value.Data.(compression.Payload)
	if !ok {
		t.Fatalf("expected compressed payload, got %T", ent.Value.Data)
	}
	if p.OriginalSize != len(text) || len(p.Data) >= len(text) {
		t.Fatalf("unexpected sizes: %#v", p)
	}
	got, err := d.Execute(context.Background(), request("GET", map[string]any{"key": "large"}))
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]any)
	if m["value"] != text {
		t.Fatalf("transparent GET mismatch")
	}
	mem, err := d.Execute(context.Background(), request("MEMORY", map[string]any{"key": "large"}))
	if err != nil {
		t.Fatal(err)
	}
	if !mem.(map[string]any)["compressed"].(bool) {
		t.Fatal("expected MEMORY to report compression")
	}
}

func TestCompressedJSONMutationIsPersisted(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	d := NewDispatcher(e)
	_ = d.SetCompression(compression.Config{Enabled: true, Algorithm: "lz4", MinBytes: 32, MinSavingsPercent: 1})
	value := map[string]any{"message": strings.Repeat("abcabcabc", 200), "version": 1}
	if _, err := d.Execute(context.Background(), request("SET", map[string]any{"key": "doc", "type": "json", "value": value})); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Execute(context.Background(), request("OP", map[string]any{"key": "doc", "operation": "SET", "args": map[string]any{"value": map[string]any{"message": strings.Repeat("xyzxyz", 200), "version": 2}}})); err != nil {
		t.Fatal(err)
	}
	ent, _ := e.GetEntry("doc")
	if _, ok := ent.Value.Data.(compression.Payload); !ok {
		t.Fatalf("JSON should remain compressed after mutation")
	}
	got, err := d.Execute(context.Background(), request("GET", map[string]any{"key": "doc"}))
	if err != nil {
		t.Fatal(err)
	}
	out := got.(map[string]any)["value"].(map[string]any)
	if fmt.Sprint(out["version"]) != "2" {
		t.Fatalf("mutation was not persisted: %#v", out)
	}
}
