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

package persistence

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"

	"github.com/newfoundcodes/amaquet/internal/protocol"
)

func TestAOFAppendReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.aof")
	a, err := Open(path, "always")
	if err != nil {
		t.Fatal(err)
	}
	req := protocol.Request{Command: "SET", Args: map[string]json.RawMessage{"key": json.RawMessage(`"x"`)}}
	if err = a.Append(req); err != nil {
		t.Fatal(err)
	}
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
	var got []protocol.Request
	n, err := Replay(path, func(r protocol.Request) error { got = append(got, r); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || len(got) != 1 || got[0].Command != "SET" {
		t.Fatalf("unexpected replay: %d %#v", n, got)
	}
}

func TestOpenMigratesPreRenameV2Header(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.aof")
	a, err := Open(path, "always")
	if err != nil {
		t.Fatal(err)
	}
	req := protocol.Request{Command: "SET", Args: map[string]json.RawMessage{"key": json.RawMessage(`"x"`)}}
	if err = a.Append(req); err != nil {
		t.Fatal(err)
	}
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.WriteAt([]byte(preRenameHeaderV2), 0); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}

	if n, replayErr := Replay(path, func(protocol.Request) error { return nil }); replayErr != nil || n != 1 {
		t.Fatalf("pre-rename replay n=%d err=%v", n, replayErr)
	}
	a, err = Open(path, "always")
	if err != nil {
		t.Fatal(err)
	}
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	header := data[:len(aofHeaderV2)]
	if string(header) != aofHeaderV2 {
		t.Fatalf("header=%q want %q", header, aofHeaderV2)
	}
}

func TestOpenMigratesPreRenameV1Journal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.aof")
	req := protocol.Request{Command: "SET", Args: map[string]json.RawMessage{"key": json.RawMessage(`"x"`)}}
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var recordHeader [8]byte
	binary.BigEndian.PutUint32(recordHeader[:4], uint32(len(payload)))
	binary.BigEndian.PutUint32(recordHeader[4:], crc32.ChecksumIEEE(payload))
	var journal bytes.Buffer
	journal.WriteString(preRenameHeaderV1)
	journal.Write(recordHeader[:])
	journal.Write(payload)
	if err = os.WriteFile(path, journal.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}

	if n, replayErr := Replay(path, func(protocol.Request) error { return nil }); replayErr != nil || n != 1 {
		t.Fatalf("pre-rename v1 replay n=%d err=%v", n, replayErr)
	}
	a, err := Open(path, "always")
	if err != nil {
		t.Fatal(err)
	}
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < len(aofHeaderV2) || string(data[:len(aofHeaderV2)]) != aofHeaderV2 {
		t.Fatalf("journal was not migrated to %q", aofHeaderV2)
	}
	if n, replayErr := Replay(path, func(got protocol.Request) error {
		if got.Command != req.Command {
			t.Fatalf("command=%q want %q", got.Command, req.Command)
		}
		return nil
	}); replayErr != nil || n != 1 {
		t.Fatalf("migrated replay n=%d err=%v", n, replayErr)
	}
}

func TestAOFIgnoresUncommittedPrepare(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.aof")
	a, err := Open(path, "always")
	if err != nil {
		t.Fatal(err)
	}
	req := protocol.Request{Command: "SET", Args: map[string]json.RawMessage{"key": json.RawMessage(`"x"`)}}
	if _, err = a.Prepare(req); err != nil {
		t.Fatal(err)
	}
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
	called := false
	n, err := Replay(path, func(protocol.Request) error { called = true; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 || called {
		t.Fatalf("uncommitted prepare replayed n=%d called=%v", n, called)
	}
}

func TestCheckpointAndCompact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.aof")
	a, err := Open(path, "always")
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"a", "b"} {
		req := protocol.Request{Command: "DEL", Args: map[string]json.RawMessage{"keys": json.RawMessage(`["` + k + `"]`)}}
		if err = a.Append(req); err != nil {
			t.Fatal(err)
		}
	}

	id, err := a.Prepare(protocol.Request{Command: "PING"})
	if err != nil {
		t.Fatal(err)
	}
	if err = a.Abort(id); err != nil {
		t.Fatal(err)
	}
	cp := filepath.Join(t.TempDir(), "checkpoint.aof")
	if err = a.Checkpoint(cp); err != nil {
		t.Fatal(err)
	}
	if n, err := Replay(cp, func(protocol.Request) error { return nil }); err != nil || n != 0 {
		t.Fatalf("compact checkpoint should represent the empty recovered state: n=%d err=%v", n, err)
	}
	before, _ := os.Stat(path)
	if err = a.Compact(); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(path)
	if after.Size() >= before.Size() {
		t.Fatalf("compact did not reduce journal: before=%d after=%d", before.Size(), after.Size())
	}
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCompactSquashesOverwrittenSetHistory(t *testing.T) {
	p := filepath.Join(t.TempDir(), "compact.aof")
	a, err := Open(p, "always")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		b, _ := json.Marshal(i)
		req := protocol.Request{Command: "SET", Args: map[string]json.RawMessage{"key": json.RawMessage(`"same"`), "type": json.RawMessage(`"integer"`), "value": b}}
		if err = a.Append(req); err != nil {
			t.Fatal(err)
		}
	}
	if err = a.Compact(); err != nil {
		t.Fatal(err)
	}
	count := 0
	if _, err = Replay(p, func(r protocol.Request) error { count++; return nil }); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one compacted request, got %d", count)
	}
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCompactSquashesCommittedBlobHistoryPerKey(t *testing.T) {
	p := filepath.Join(t.TempDir(), "blob.aof")
	a, err := Open(p, "always")
	if err != nil {
		t.Fatal(err)
	}
	seq := func(id, data string) {
		for _, r := range []protocol.Request{
			{Command: "BLOB_BEGIN", Args: map[string]json.RawMessage{"key": json.RawMessage(`"blob"`), "upload_id": json.RawMessage(`"` + id + `"`), "total_size": json.RawMessage(`3`)}},
			{Command: "BLOB_CHUNK", Args: map[string]json.RawMessage{"upload_id": json.RawMessage(`"` + id + `"`), "offset": json.RawMessage(`0`), "data": json.RawMessage(`"` + data + `"`)}},
			{Command: "BLOB_COMMIT", Args: map[string]json.RawMessage{"upload_id": json.RawMessage(`"` + id + `"`)}},
		} {
			if err := a.Append(r); err != nil {
				t.Fatal(err)
			}
		}
	}
	seq("u1", "YWJj")
	seq("u2", "eHl6")
	if err = a.Compact(); err != nil {
		t.Fatal(err)
	}
	var got []protocol.Request
	if _, err = Replay(p, func(r protocol.Request) error { got = append(got, r); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d compacted blob requests, want 3", len(got))
	}
	var id string
	_ = json.Unmarshal(got[0].Args["upload_id"], &id)
	if id != "u2" {
		t.Fatalf("kept upload %q, want u2", id)
	}
	_ = a.Close()
}

func TestRestoreCheckpointValidatesAndAtomicallyInstalls(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.aof")
	a, err := Open(src, "always")
	if err != nil {
		t.Fatal(err)
	}
	r := protocol.Request{Command: "SET", Args: map[string]json.RawMessage{"key": json.RawMessage(`"x"`), "type": json.RawMessage(`"integer"`), "value": json.RawMessage(`7`)}}
	if err = a.Append(r); err != nil {
		t.Fatal(err)
	}
	cp := filepath.Join(dir, "cp.aof")
	if err = a.Checkpoint(cp); err != nil {
		t.Fatal(err)
	}
	_ = a.Close()
	dst := filepath.Join(dir, "restored.aof")
	if err = RestoreCheckpoint(cp, dst); err != nil {
		t.Fatal(err)
	}
	n, err := Replay(dst, func(protocol.Request) error { return nil })
	if err != nil || n != 1 {
		t.Fatalf("restore replay n=%d err=%v", n, err)
	}
	bad := filepath.Join(dir, "bad.aof")
	if err = os.WriteFile(bad, []byte("garbage"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = RestoreCheckpoint(bad, dst); err == nil {
		t.Fatal("corrupt checkpoint was accepted")
	}
}
