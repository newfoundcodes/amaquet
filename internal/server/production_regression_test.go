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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/newfoundcodes/amaquet/internal/core"
	"github.com/newfoundcodes/amaquet/internal/protocol"
	nt "github.com/newfoundcodes/amaquet/internal/types"
)

func req(command string, args map[string]any) protocol.Request {
	out := protocol.Request{Command: command, Args: map[string]json.RawMessage{}}
	for k, v := range args {
		b, _ := json.Marshal(v)
		out.Args[k] = b
	}
	return out
}

func TestCompositeMutationPreflightPreventsMemoryOvershoot(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	d := NewDispatcher(e)
	defer d.Close()
	if err := e.ConfigureMemory(16<<10, "noeviction"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Execute(context.Background(), req("CREATE", map[string]any{"key": "bits", "type": "bitmap"})); err != nil {
		t.Fatal(err)
	}
	before := e.MemoryUsage()
	_, err := d.Execute(context.Background(), req("OP", map[string]any{"key": "bits", "operation": "SET", "args": map[string]any{"bit": uint64(8 << 20), "value": true}}))
	if !errors.Is(err, core.ErrMemoryLimit) {
		t.Fatalf("expected memory limit, got %v", err)
	}
	if got := e.MemoryUsage(); got != before {
		t.Fatalf("mutation changed accounted memory: before=%d after=%d", before, got)
	}
}

func TestUploadOwnedAndCleanedWithoutRAMBuffer(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	d := NewDispatcher(e)
	defer d.Close()
	dir := t.TempDir()
	if err := d.ConfigureUploads(dir, time.Second, 1<<20); err != nil {
		t.Fatal(err)
	}
	ctx := withUploadOwner(context.Background(), "conn-a")
	out, err := d.Execute(ctx, req("BLOB_BEGIN", map[string]any{"key": "blob", "total_size": 1024, "upload_id": "u", "connection_scoped": true}))
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	d.uploadMu.Lock()
	u := d.uploads["u"]
	d.uploadMu.Unlock()
	if u == nil {
		t.Fatal("upload missing")
	}
	if u.total != 1024 {
		t.Fatalf("total=%d", u.total)
	}
	if _, err := os.Stat(u.path); err != nil {
		t.Fatal(err)
	}
	d.AbortUploadsForOwner("conn-a")
	if _, err := os.Stat(u.path); !os.IsNotExist(err) {
		t.Fatalf("temporary upload not removed: %v", err)
	}
}

type failingCommitLog struct {
	unhealthy error
	next      uint64
}

func (f *failingCommitLog) Prepare(protocol.Request) (uint64, error) {
	if f.unhealthy != nil {
		return 0, f.unhealthy
	}
	f.next++
	return f.next, nil
}
func (f *failingCommitLog) Commit(uint64) error {
	f.unhealthy = errors.New("disk failed")
	return f.unhealthy
}
func (f *failingCommitLog) Abort(uint64) error { return nil }
func (f *failingCommitLog) HealthError() error { return f.unhealthy }

func TestCommitFailureEntersWriteFailStopMode(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	d := NewDispatcher(e)
	defer d.Close()
	log := &failingCommitLog{}
	d.SetMutationLog(log)
	if _, err := d.Execute(context.Background(), req("SET", map[string]any{"key": "first", "type": "integer", "value": 1})); err == nil {
		t.Fatal("expected commit failure")
	}
	if !e.Exists("first") {
		t.Fatal("first mutation should reflect acknowledged fail-stop divergence")
	}
	if _, err := d.Execute(context.Background(), req("SET", map[string]any{"key": "second", "type": "integer", "value": 2})); err == nil {
		t.Fatal("expected persistence-unavailable rejection")
	}
	if e.Exists("second") {
		t.Fatal("write occurred after persistence entered fail-stop mode")
	}
}

func TestResumableUploadSurvivesOwnerDisconnect(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	d := NewDispatcher(e)
	defer d.Close()
	if err := d.ConfigureUploads(t.TempDir(), time.Minute, 1<<20); err != nil {
		t.Fatal(err)
	}
	ctxA := withUploadOwner(context.Background(), "conn-a")
	if _, err := d.Execute(ctxA, req("BLOB_BEGIN", map[string]any{"key": "blob", "total_size": 5, "upload_id": "resume"})); err != nil {
		t.Fatal(err)
	}
	d.AbortUploadsForOwner("conn-a")
	ctxB := withUploadOwner(context.Background(), "conn-b")
	if _, err := d.Execute(ctxB, req("BLOB_CHUNK", map[string]any{"upload_id": "resume", "offset": 0, "data": "aGVsbG8="})); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Execute(ctxB, req("BLOB_COMMIT", map[string]any{"upload_id": "resume"})); err != nil {
		t.Fatal(err)
	}
	v, _, err := e.Get("blob")
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != core.TypeBinaryString {
		t.Fatalf("type=%s", v.Type)
	}
}

func TestBlobRejectsZeroLengthAndWrongOwner(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	d := NewDispatcher(e)
	defer d.Close()
	if err := d.ConfigureUploads(t.TempDir(), time.Minute, 1<<20); err != nil {
		t.Fatal(err)
	}
	ctxA := withUploadOwner(context.Background(), "A")
	if _, err := d.Execute(ctxA, req("BLOB_BEGIN", map[string]any{"key": "b", "upload_id": "u", "total_size": 4, "connection_scoped": true})); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Execute(ctxA, req("BLOB_CHUNK", map[string]any{"upload_id": "u", "offset": 0, "data": ""})); err == nil {
		t.Fatal("zero-length chunk accepted")
	}
	if _, err := d.Execute(context.Background(), req("BLOB_CHUNK", map[string]any{"upload_id": "u", "offset": 0, "data": "YWJjZA=="})); err == nil {
		t.Fatal("ownerless context bypassed connection ownership")
	}
}

func TestBlobCommitUsesChunkedMemoryAndRangeRead(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	d := NewDispatcher(e)
	defer d.Close()
	if err := d.ConfigureUploads(t.TempDir(), time.Minute, 8<<20); err != nil {
		t.Fatal(err)
	}
	payload := make([]byte, 3*1024*1024+17)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	if _, err := d.Execute(context.Background(), req("BLOB_BEGIN", map[string]any{"key": "big", "upload_id": "big-u", "total_size": len(payload)})); err != nil {
		t.Fatal(err)
	}
	for off := 0; off < len(payload); off += 512 * 1024 {
		end := off + 512*1024
		if end > len(payload) {
			end = len(payload)
		}
		enc := base64.StdEncoding.EncodeToString(payload[off:end])
		if _, err := d.Execute(context.Background(), req("BLOB_CHUNK", map[string]any{"upload_id": "big-u", "offset": off, "data": enc})); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.Execute(context.Background(), req("BLOB_COMMIT", map[string]any{"upload_id": "big-u"})); err != nil {
		t.Fatal(err)
	}
	v, _, err := e.Get("big")
	if err != nil {
		t.Fatal(err)
	}
	cb, ok := v.Data.(*nt.ChunkedBinary)
	if !ok {
		t.Fatalf("stored value is %T, want *ChunkedBinary", v.Data)
	}
	if cb.Size != int64(len(payload)) {
		t.Fatalf("size=%d", cb.Size)
	}
	out, err := d.Execute(context.Background(), req("BLOB_READ", map[string]any{"key": "big", "offset": 1024, "length": 4096}))
	if err != nil {
		t.Fatal(err)
	}
	m := out.(map[string]any)
	got, err := base64.StdEncoding.DecodeString(m["data"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload[1024:1024+4096]) {
		t.Fatal("range mismatch")
	}
}
