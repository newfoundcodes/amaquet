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
	"path/filepath"
	"testing"
	"time"

	"github.com/newfoundcodes/amaquet/internal/core"
	"github.com/newfoundcodes/amaquet/internal/persistence"
	"github.com/newfoundcodes/amaquet/internal/protocol"
	nt "github.com/newfoundcodes/amaquet/internal/types"
)

func TestAOFReplayPreservesAbsoluteExpiry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.aof")
	e := core.NewEngine()
	d := NewDispatcher(e)
	a, err := persistence.Open(path, "always")
	if err != nil {
		t.Fatal(err)
	}
	d.SetMutationLog(a)
	req := protocol.Request{Command: "SET", Args: map[string]json.RawMessage{}}
	for k, v := range map[string]any{"key": "ttl", "type": "utf8_string", "value": "x", "ttl_ms": int64(60)} {
		b, _ := json.Marshal(v)
		req.Args[k] = b
	}
	if _, err = d.Execute(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
	_ = e.Close()
	time.Sleep(90 * time.Millisecond)
	e2 := core.NewEngine()
	defer e2.Close()
	d2 := NewDispatcher(e2)
	_, err = persistence.Replay(path, func(r protocol.Request) error { _, e := d2.ExecuteReplay(context.Background(), r); return e })
	if err != nil {
		t.Fatal(err)
	}
	if e2.Exists("ttl") {
		t.Fatal("expired key was resurrected with a fresh TTL during replay")
	}
}

func TestChunkedBlobUpload(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	d := NewDispatcher(e)
	begin := protocol.Request{Command: "BLOB_BEGIN", Args: map[string]json.RawMessage{}}
	for k, v := range map[string]any{"key": "blob", "upload_id": "u1", "total_size": 6} {
		b, _ := json.Marshal(v)
		begin.Args[k] = b
	}
	if _, err := d.Execute(context.Background(), begin); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		off  int
		data string
	}{{0, "YWJj"}, {3, "ZGVm"}} {
		req := protocol.Request{Command: "BLOB_CHUNK", Args: map[string]json.RawMessage{}}
		for k, v := range map[string]any{"upload_id": "u1", "offset": c.off, "data": c.data} {
			b, _ := json.Marshal(v)
			req.Args[k] = b
		}
		if _, err := d.Execute(context.Background(), req); err != nil {
			t.Fatal(err)
		}
	}
	commit := protocol.Request{Command: "BLOB_COMMIT", Args: map[string]json.RawMessage{"upload_id": json.RawMessage(`"u1"`)}}
	if _, err := d.Execute(context.Background(), commit); err != nil {
		t.Fatal(err)
	}
	got, err := d.Execute(context.Background(), protocol.Request{Command: "GET", Args: map[string]json.RawMessage{"key": json.RawMessage(`"blob"`)}})
	if err != nil {
		t.Fatal(err)
	}
	m := got.(map[string]any)
	if m["value"] != "YWJjZGVm" {
		t.Fatalf("got %#v", m)
	}
}

func TestAOFReplayPreservesStreamID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stream.aof")
	e := core.NewEngine()
	d := NewDispatcher(e)
	a, err := persistence.Open(path, "always")
	if err != nil {
		t.Fatal(err)
	}
	d.SetMutationLog(a)
	mk := func(cmd string, m map[string]any) protocol.Request {
		r := protocol.Request{Command: cmd, Args: map[string]json.RawMessage{}}
		for k, v := range m {
			b, _ := json.Marshal(v)
			r.Args[k] = b
		}
		return r
	}
	if _, err = d.Execute(context.Background(), mk("CREATE", map[string]any{"key": "s", "type": "stream"})); err != nil {
		t.Fatal(err)
	}
	firstAny, err := d.Execute(context.Background(), mk("OP", map[string]any{"key": "s", "operation": "ADD", "args": map[string]any{"fields": map[string]any{"x": map[string]any{"type": "utf8_string", "value": "y"}}}}))
	if err != nil {
		t.Fatal(err)
	}
	first := firstAny.(nt.StreamID)
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
	_ = e.Close()
	e2 := core.NewEngine()
	defer e2.Close()
	d2 := NewDispatcher(e2)
	if _, err = persistence.Replay(path, func(r protocol.Request) error { _, e := d2.ExecuteReplay(context.Background(), r); return e }); err != nil {
		t.Fatal(err)
	}
	gotAny, err := d2.Execute(context.Background(), mk("OP", map[string]any{"key": "s", "operation": "RANGE", "args": map[string]any{"count": 10}}))
	if err != nil {
		t.Fatal(err)
	}
	entries := gotAny.([]nt.StreamEntry)
	if len(entries) != 1 || entries[0].ID != first {
		t.Fatalf("stream id changed: first=%v replay=%v", first, entries)
	}
}

func TestAOFReplayCASPreservesAbsoluteExpiry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cas.aof")
	e := core.NewEngine()
	d := NewDispatcher(e)
	a, err := persistence.Open(path, "always")
	if err != nil {
		t.Fatal(err)
	}
	d.SetMutationLog(a)
	mk := func(cmd string, m map[string]any) protocol.Request {
		r := protocol.Request{Command: cmd, Args: map[string]json.RawMessage{}}
		for k, v := range m {
			b, _ := json.Marshal(v)
			r.Args[k] = b
		}
		return r
	}
	if _, err = d.Execute(context.Background(), mk("SET", map[string]any{"key": "cas-ttl", "type": "integer", "value": 1})); err != nil {
		t.Fatal(err)
	}
	if _, err = d.Execute(context.Background(), mk("CAS", map[string]any{"key": "cas-ttl", "expected_version": 1, "type": "integer", "value": 2, "ttl_ms": int64(60)})); err != nil {
		t.Fatal(err)
	}
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
	_ = e.Close()
	time.Sleep(90 * time.Millisecond)
	e2 := core.NewEngine()
	defer e2.Close()
	d2 := NewDispatcher(e2)
	if _, err = persistence.Replay(path, func(r protocol.Request) error { _, e := d2.ExecuteReplay(context.Background(), r); return e }); err != nil {
		t.Fatal(err)
	}
	if e2.Exists("cas-ttl") {
		t.Fatal("expired CAS result was resurrected during replay")
	}
}
