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
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/newfoundcodes/amaquet/internal/compression"
	"github.com/newfoundcodes/amaquet/internal/core"
	"github.com/newfoundcodes/amaquet/internal/protocol"
	nt "github.com/newfoundcodes/amaquet/internal/types"
)

type MutationLogger interface {
	Prepare(protocol.Request) (uint64, error)
	Commit(uint64) error
	Abort(uint64) error
	HealthError() error
}

type uploadSpan struct{ start, end int64 }

const maxUploadSpans = 65536
const maxBlobReadBytes = 32 << 20

type blobUpload struct {
	key            string
	path           string
	total          int64
	spans          []uploadSpan
	received       int64
	expiresAt      time.Time
	uploadDeadline time.Time
	owner          string
	committing     bool
	nx, xx         bool
}

type uploadOwnerContextKey struct{}

func withUploadOwner(ctx context.Context, owner string) context.Context {
	return context.WithValue(ctx, uploadOwnerContextKey{}, owner)
}
func uploadOwner(ctx context.Context) string {
	v, _ := ctx.Value(uploadOwnerContextKey{}).(string)
	return v
}

type Dispatcher struct {
	Engine                *core.Engine
	StartedAt             time.Time
	mutationLog           MutationLogger
	compressor            *compression.Manager
	uploadMu              sync.Mutex
	uploads               map[string]*blobUpload
	pendingUploadBytes    int64
	maxPendingUploadBytes int64
	uploadTTL             time.Duration
	uploadDir             string
	uploadStop            chan struct{}
	uploadDone            chan struct{}
	mutationMu            sync.Mutex
}

func NewDispatcher(e *core.Engine) *Dispatcher {
	m, _ := compression.NewManager(compression.DefaultConfig())
	dir := filepath.Join(os.TempDir(), "amaquet-uploads")
	_ = os.MkdirAll(dir, 0700)
	d := &Dispatcher{Engine: e, StartedAt: time.Now().UTC(), compressor: m, uploads: map[string]*blobUpload{}, maxPendingUploadBytes: 1 << 30, uploadTTL: 10 * time.Minute, uploadDir: dir, uploadStop: make(chan struct{}), uploadDone: make(chan struct{})}
	go d.uploadSweeper()
	return d
}

func (d *Dispatcher) ConfigureUploads(dir string, ttl time.Duration, maxPending int64) error {
	if ttl < time.Second || maxPending < 1 {
		return core.ErrInvalidArgument
	}
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "amaquet-uploads")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	if stale, _ := filepath.Glob(filepath.Join(dir, "upload-*.part")); len(stale) > 0 {
		for _, name := range stale {
			_ = os.Remove(name)
		}
	}
	d.uploadMu.Lock()
	d.uploadDir = dir
	d.uploadTTL = ttl
	d.maxPendingUploadBytes = maxPending
	d.uploadMu.Unlock()
	return nil
}
func (d *Dispatcher) uploadSweeper() {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	defer close(d.uploadDone)
	for {
		select {
		case now := <-t.C:
			d.cleanupExpiredUploads(now)
		case <-d.uploadStop:
			return
		}
	}
}
func (d *Dispatcher) cleanupExpiredUploads(now time.Time) {
	d.uploadMu.Lock()
	var paths []string
	for id, u := range d.uploads {
		if !u.uploadDeadline.IsZero() && !now.Before(u.uploadDeadline) {
			delete(d.uploads, id)
			d.pendingUploadBytes -= u.total
			paths = append(paths, u.path)
		}
	}
	if d.pendingUploadBytes < 0 {
		d.pendingUploadBytes = 0
	}
	d.uploadMu.Unlock()
	for _, p := range paths {
		_ = os.Remove(p)
	}
}
func (d *Dispatcher) AbortUploadsForOwner(owner string) {
	if owner == "" {
		return
	}
	d.uploadMu.Lock()
	var paths []string
	for id, u := range d.uploads {
		if u.owner == owner {
			delete(d.uploads, id)
			d.pendingUploadBytes -= u.total
			paths = append(paths, u.path)
		}
	}
	if d.pendingUploadBytes < 0 {
		d.pendingUploadBytes = 0
	}
	d.uploadMu.Unlock()
	for _, p := range paths {
		_ = os.Remove(p)
	}
}
func (d *Dispatcher) DiscardIncompleteUploads() {
	d.uploadMu.Lock()
	var paths []string
	for id, u := range d.uploads {
		delete(d.uploads, id)
		d.pendingUploadBytes -= u.total
		paths = append(paths, u.path)
	}
	if d.pendingUploadBytes < 0 {
		d.pendingUploadBytes = 0
	}
	d.uploadMu.Unlock()
	for _, p := range paths {
		_ = os.Remove(p)
	}
}
func (d *Dispatcher) Close() {
	select {
	case <-d.uploadStop:
	default:
		close(d.uploadStop)
	}
	<-d.uploadDone
	d.DiscardIncompleteUploads()
}

func (d *Dispatcher) SetCompression(cfg compression.Config) error {
	m, err := compression.NewManager(cfg)
	if err != nil {
		return err
	}
	d.compressor = m
	return nil
}

func (d *Dispatcher) SetMutationLog(log MutationLogger) { d.mutationLog = log }

func (d *Dispatcher) WithMutationBarrier(fn func() error) error {
	d.mutationMu.Lock()
	defer d.mutationMu.Unlock()
	return fn()
}

func (d *Dispatcher) Execute(ctx context.Context, req protocol.Request) (any, error) {
	if d.mutationLog != nil && potentialJournal(req) {
		d.mutationMu.Lock()
		defer d.mutationMu.Unlock()
	}
	journalReq := d.resolveJournalRequest(req)
	var txID uint64
	if d.mutationLog != nil && d.shouldJournal(journalReq) {
		if err := d.mutationLog.HealthError(); err != nil {
			return nil, fmt.Errorf("persistence unavailable: %w", err)
		}
		var err error
		txID, err = d.mutationLog.Prepare(journalReq)
		if err != nil {
			return nil, fmt.Errorf("AOF prepare failed: %w", err)
		}
	}
	result, err := d.execute(ctx, journalReq)
	if err != nil {
		if txID != 0 {
			_ = d.mutationLog.Abort(txID)
		}
		return nil, err
	}
	if txID != 0 {
		if err := d.mutationLog.Commit(txID); err != nil {
			return nil, fmt.Errorf("mutation applied but AOF commit failed; persistence is degraded: %w", err)
		}
	}
	return result, nil
}

func cloneRequest(req protocol.Request) protocol.Request {
	cp := protocol.Request{Command: req.Command}
	if req.Args != nil {
		cp.Args = make(map[string]json.RawMessage, len(req.Args))
		for k, v := range req.Args {
			cp.Args[k] = append(json.RawMessage(nil), v...)
		}
	}
	return cp
}
func setRawArg(req *protocol.Request, key string, value any) {
	if req.Args == nil {
		req.Args = map[string]json.RawMessage{}
	}
	b, _ := json.Marshal(value)
	req.Args[key] = b
}
func potentialJournal(req protocol.Request) bool {
	cmd := strings.ToUpper(strings.TrimSpace(req.Command))
	switch cmd {
	case "SET", "CREATE", "CAS", "DEL", "EXPIRE", "PERSIST", "BLOB_BEGIN", "BLOB_CHUNK", "BLOB_COMMIT", "BLOB_ABORT":
		return true
	case "OP":
		a := args(req.Args)
		op := strings.ToUpper(a.optString("operation", ""))
		return op != "" && !isReadOp(op)
	}
	return false
}
func opArgsMap(req protocol.Request) map[string]json.RawMessage {
	var m map[string]json.RawMessage
	if raw := req.Args["args"]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &m)
	}
	if m == nil {
		m = map[string]json.RawMessage{}
	}
	return m
}
func setOpArg(req *protocol.Request, key string, value any) {
	m := opArgsMap(*req)
	b, _ := json.Marshal(value)
	m[key] = b
	outer, _ := json.Marshal(m)
	if req.Args == nil {
		req.Args = map[string]json.RawMessage{}
	}
	req.Args["args"] = outer
}

func (d *Dispatcher) resolveJournalRequest(req protocol.Request) protocol.Request {
	r := cloneRequest(req)
	a := args(r.Args)
	cmd := strings.ToUpper(strings.TrimSpace(r.Command))
	now := time.Now().UTC()
	switch cmd {
	case "SET", "CREATE", "CAS":
		if ttl := a.duration("ttl_ms", 0); ttl > 0 {
			setRawArg(&r, "expires_at_unix_nano", now.Add(ttl).UnixNano())
		}
	case "BLOB_BEGIN":
		if a.optString("upload_id", "") == "" {
			var b [16]byte
			_, _ = rand.Read(b[:])
			setRawArg(&r, "upload_id", hex.EncodeToString(b[:]))
		}
		if ttl := a.duration("ttl_ms", 0); ttl > 0 {
			setRawArg(&r, "expires_at_unix_nano", now.Add(ttl).UnixNano())
		}
	case "EXPIRE":
		if ttl := a.duration("ttl_ms", 0); ttl > 0 {
			setRawArg(&r, "expires_at_unix_nano", now.Add(ttl).UnixNano())
		}
	case "OP":
		op := strings.ToUpper(a.optString("operation", ""))
		key := a.optString("key", "")
		oa := args(opArgsMap(r))
		t, err := d.Engine.Type(key)
		if err == nil {
			switch t {
			case core.TypeTimeSeries, core.TypeCounterSeries, core.TypeGaugeSeries:
				if (op == "ADD" || op == "SET") && len(oa.raw("timestamp")) == 0 {
					setOpArg(&r, "timestamp", now.UnixMilli())
				}
			case core.TypeStream:
				if op == "ADD" && oa.optString("id", "") == "" {
					if v, _, e := d.Engine.Get(key); e == nil {
						if st, ok := v.Data.(*nt.Stream); ok {
							setOpArg(&r, "id", st.NextIDAt(now.UnixMilli()).String())
						}
					}
				}
			case core.TypePersistentTopic:
				if op == "PUBLISH" && len(oa.raw("timestamp_unix_nano")) == 0 {
					setOpArg(&r, "timestamp_unix_nano", now.UnixNano())
				}
			case core.TypeEventLog:
				if op == "APPEND" && len(oa.raw("timestamp_unix_nano")) == 0 {
					setOpArg(&r, "timestamp_unix_nano", now.UnixNano())
				}
			case core.TypeDelayedQueue:
				if op == "POP_READY" && len(oa.raw("now_unix_nano")) == 0 {
					setOpArg(&r, "now_unix_nano", now.UnixNano())
				}
			case core.TypeReliableQueue:
				if (op == "ENQUEUE" || op == "CLAIM") && len(oa.raw("now_unix_nano")) == 0 {
					setOpArg(&r, "now_unix_nano", now.UnixNano())
				}
			}
		}

	}
	return r
}

func (d *Dispatcher) ExecuteReplay(ctx context.Context, req protocol.Request) (any, error) {
	return d.execute(ctx, req)
}

func (d *Dispatcher) shouldJournal(req protocol.Request) bool {
	cmd := strings.ToUpper(strings.TrimSpace(req.Command))
	switch cmd {
	case "SET", "CREATE", "CAS", "DEL", "EXPIRE", "PERSIST", "BLOB_BEGIN", "BLOB_CHUNK", "BLOB_COMMIT", "BLOB_ABORT":
		return true
	case "OP":
		a := args(req.Args)
		op := strings.ToUpper(a.optString("operation", ""))
		if op == "" || isReadOp(op) {
			return false
		}
		key := a.optString("key", "")
		if key == "" {
			return false
		}
		t, err := d.Engine.Type(key)
		if err != nil {
			return false
		}

		switch t {
		case core.TypePubSub, core.TypeLease, core.TypeLock, core.TypeBarrier, core.TypeSemaphore, core.TypeRateLimitBucket:
			return false
		}
		return true
	}
	return false
}

func (d *Dispatcher) execute(ctx context.Context, req protocol.Request) (any, error) {
	cmd := strings.ToUpper(strings.TrimSpace(req.Command))
	a := args(req.Args)
	switch cmd {
	case "PING":
		return "PONG", nil
	case "SET":
		return d.set(a)
	case "CAS":
		return d.cas(a)
	case "BATCH":
		return d.batch(ctx, a)
	case "CREATE":
		return d.create(a)
	case "GET":
		return d.get(a)
	case "DEL":
		keys, e := a.strings("keys")
		if e != nil {
			return nil, e
		}
		return d.Engine.Delete(keys...), nil
	case "EXISTS":
		k, e := a.string("key")
		if e != nil {
			return nil, e
		}
		return d.Engine.Exists(k), nil
	case "TYPE":
		k, e := a.string("key")
		if e != nil {
			return nil, e
		}
		t, e := d.Engine.Type(k)
		if e != nil {
			return nil, e
		}
		return t.String(), nil
	case "TTL":
		k, e := a.string("key")
		if e != nil {
			return nil, e
		}
		ttl, e := d.Engine.TTL(k)
		if e != nil {
			return nil, e
		}
		if ttl < 0 {
			return int64(-1), nil
		}
		return ttl.Milliseconds(), nil
	case "EXPIRE":
		k, e := a.string("key")
		if e != nil {
			return nil, e
		}
		if at := a.int64Default("expires_at_unix_nano", 0); at > 0 {
			return true, d.Engine.ExpireAt(k, time.Unix(0, at).UTC())
		}
		ttl := a.duration("ttl_ms", 0)
		if ttl <= 0 {
			return nil, fmt.Errorf("ttl must be positive")
		}
		return true, d.Engine.Expire(k, ttl)
	case "PERSIST":
		k, e := a.string("key")
		if e != nil {
			return nil, e
		}
		return true, d.Engine.Persist(k)
	case "KEYS":
		return d.Engine.Keys(a.optString("prefix", ""), rawToInt(a.raw("limit"), 1000)), nil
	case "SCAN":
		keys, next := d.Engine.Scan(a.optString("cursor", ""), a.optString("prefix", ""), rawToInt(a.raw("count"), 100))
		return map[string]any{"keys": keys, "cursor": next}, nil
	case "BLOB_BEGIN":
		return d.blobBegin(ctx, a)
	case "BLOB_CHUNK":
		return d.blobChunk(ctx, a)
	case "BLOB_COMMIT":
		return d.blobCommit(ctx, a)
	case "BLOB_READ":
		return d.blobRead(a)
	case "BLOB_ABORT":
		return d.blobAbort(ctx, a)
	case "TYPES":
		names := make([]string, 0)
		for _, t := range core.AllDataTypes() {
			names = append(names, t.String())
		}
		return names, nil
	case "INFO":
		return map[string]any{"name": "Amaquet", "protocol_version": protocol.Version, "keys": d.Engine.Len(), "uptime_seconds": int64(time.Since(d.StartedAt).Seconds()), "types": len(core.AllDataTypes()), "compression": d.Engine.CompressionStats(), "memory": d.Engine.MemoryStats()}, nil
	case "MEMORY":
		k, e := a.string("key")
		if e != nil {
			return nil, e
		}
		ent, e := d.Engine.GetEntry(k)
		if e != nil {
			return nil, e
		}
		out := map[string]any{"key": k, "type": ent.Value.Type.String(), "version": ent.Version, "access_count": ent.AccessCount(), "approx_bytes": ent.ApproxBytes(), "compressed": false}
		if info, ok := compressionPayloadInfo(ent.Value); ok {
			for x, y := range info {
				out[x] = y
			}
		}
		return out, nil
	case "OP":
		return d.op(ctx, a)
	default:
		return nil, fmt.Errorf("unknown command %q", cmd)
	}
}

func (d *Dispatcher) cas(a args) (any, error) {
	k, err := a.string("key")
	if err != nil {
		return nil, err
	}
	tn, err := a.string("type")
	if err != nil {
		return nil, err
	}
	t, err := core.ParseDataType(tn)
	if err != nil {
		return nil, err
	}
	v, err := decodeTypedValue(t, a.raw("value"))
	if err != nil {
		return nil, err
	}
	v, err = d.prepareValue(v)
	if err != nil {
		return nil, err
	}
	expected := a.uint64Default("expected_version", 0)
	if expected == 0 {
		return nil, fmt.Errorf("expected_version must be positive")
	}
	if at := a.int64Default("expires_at_unix_nano", 0); at > 0 {
		err = d.Engine.CompareAndSetAbsolute(k, expected, v, time.Unix(0, at).UTC())
	} else {
		err = d.Engine.CompareAndSet(k, expected, v, a.duration("ttl_ms", 0))
	}
	if err != nil {
		return nil, err
	}
	return true, nil
}
func (d *Dispatcher) batch(ctx context.Context, a args) (any, error) {
	raw := a.raw("requests")
	if len(raw) == 0 {
		return nil, fmt.Errorf("requests is required")
	}
	var reqs []protocol.Request
	if err := json.Unmarshal(raw, &reqs); err != nil {
		return nil, err
	}
	if len(reqs) > 1024 {
		return nil, fmt.Errorf("batch exceeds 1024 requests")
	}
	cont := a.bool("continue_on_error", false)
	out := make([]map[string]any, 0, len(reqs))
	for _, r := range reqs {
		res, err := d.Execute(ctx, r)
		item := map[string]any{"ok": err == nil}
		if err != nil {
			item["error"] = err.Error()
			out = append(out, item)
			if !cont {
				break
			}
			continue
		}
		item["result"] = res
		out = append(out, item)
	}
	return out, nil
}

func (d *Dispatcher) blobBegin(ctx context.Context, a args) (any, error) {
	key, err := a.string("key")
	if err != nil {
		return nil, err
	}
	id, err := a.string("upload_id")
	if err != nil {
		return nil, err
	}
	total := a.int64Default("total_size", -1)
	if total < 0 || total > 1<<30 {
		return nil, fmt.Errorf("total_size must be between 0 and 1 GiB")
	}
	d.uploadMu.Lock()
	if _, ok := d.uploads[id]; ok {
		d.uploadMu.Unlock()
		return nil, core.ErrExists
	}
	if total > d.maxPendingUploadBytes-d.pendingUploadBytes {
		d.uploadMu.Unlock()
		return nil, fmt.Errorf("pending blob uploads exceed %d bytes", d.maxPendingUploadBytes)
	}
	dir, ttl := d.uploadDir, d.uploadTTL
	d.uploadMu.Unlock()
	f, err := os.CreateTemp(dir, "upload-*.part")
	if err != nil {
		return nil, err
	}
	path := f.Name()
	if err = f.Truncate(total); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, err
	}
	if err = f.Close(); err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	owner := ""
	if a.bool("connection_scoped", false) {
		owner = uploadOwner(ctx)
	}
	u := &blobUpload{key: key, path: path, total: total, nx: a.bool("nx", false), xx: a.bool("xx", false), owner: owner, uploadDeadline: time.Now().Add(ttl)}
	if at := a.int64Default("expires_at_unix_nano", 0); at > 0 {
		u.expiresAt = time.Unix(0, at).UTC()
	}
	d.uploadMu.Lock()
	if _, ok := d.uploads[id]; ok {
		d.uploadMu.Unlock()
		_ = os.Remove(path)
		return nil, core.ErrExists
	}
	d.pendingUploadBytes += total
	d.uploads[id] = u
	d.uploadMu.Unlock()
	return map[string]any{"upload_id": id, "total_size": total, "expires_in_ms": ttl.Milliseconds(), "connection_scoped": owner != ""}, nil
}
func addUploadSpan(spans []uploadSpan, start, end int64) ([]uploadSpan, int64, error) {
	if start < 0 || end <= start {
		return spans, 0, core.ErrInvalidArgument
	}
	if len(spans) >= maxUploadSpans {
		return spans, 0, fmt.Errorf("upload chunk limit exceeded")
	}
	i := sort.Search(len(spans), func(i int) bool { return spans[i].start >= start })
	if i > 0 && start < spans[i-1].end {
		return spans, 0, fmt.Errorf("chunk overlaps existing range")
	}
	if i < len(spans) && end > spans[i].start {
		return spans, 0, fmt.Errorf("chunk overlaps existing range")
	}
	spans = append(spans, uploadSpan{})
	copy(spans[i+1:], spans[i:])
	spans[i] = uploadSpan{start, end}
	return spans, end - start, nil
}
func (d *Dispatcher) blobChunk(ctx context.Context, a args) (any, error) {
	id, err := a.string("upload_id")
	if err != nil {
		return nil, err
	}
	off := a.int64Default("offset", -1)
	if off < 0 {
		return nil, core.ErrInvalidArgument
	}
	enc, err := a.string("data")
	if err != nil {
		return nil, err
	}
	b, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 chunk: %w", err)
	}
	d.uploadMu.Lock()
	u, ok := d.uploads[id]
	if !ok {
		d.uploadMu.Unlock()
		return nil, core.ErrNotFound
	}
	if u.owner != "" && u.owner != uploadOwner(ctx) {
		d.uploadMu.Unlock()
		return nil, fmt.Errorf("upload belongs to another connection")
	}
	end := off + int64(len(b))
	if end > u.total {
		d.uploadMu.Unlock()
		return nil, fmt.Errorf("chunk exceeds declared total_size")
	}
	spans, added, err := addUploadSpan(u.spans, off, end)
	if err != nil {
		d.uploadMu.Unlock()
		return nil, err
	}
	path := u.path
	ttl := d.uploadTTL
	d.uploadMu.Unlock()
	f, err := os.OpenFile(path, os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	_, err = f.WriteAt(b, off)
	cerr := f.Close()
	if err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}
	d.uploadMu.Lock()
	if cur := d.uploads[id]; cur == u {
		u.spans = spans
		u.received += added
		u.uploadDeadline = time.Now().Add(ttl)
	}
	rec, total := u.received, u.total
	d.uploadMu.Unlock()
	return map[string]any{"received": rec, "total_size": total}, nil
}
func (d *Dispatcher) blobCommit(ctx context.Context, a args) (any, error) {
	id, err := a.string("upload_id")
	if err != nil {
		return nil, err
	}
	d.uploadMu.Lock()
	u, ok := d.uploads[id]
	if !ok {
		d.uploadMu.Unlock()
		return nil, core.ErrNotFound
	}
	if u.owner != "" && u.owner != uploadOwner(ctx) {
		d.uploadMu.Unlock()
		return nil, fmt.Errorf("upload belongs to another connection")
	}
	if u.committing {
		d.uploadMu.Unlock()
		return nil, core.ErrConflict
	}
	if u.received != u.total {
		d.uploadMu.Unlock()
		return nil, fmt.Errorf("upload incomplete: received %d of %d", u.received, u.total)
	}
	u.committing = true
	u.uploadDeadline = time.Time{}
	ttl := d.uploadTTL
	d.uploadMu.Unlock()
	fail := func(err error) (any, error) {
		d.uploadMu.Lock()
		if cur := d.uploads[id]; cur == u {
			u.committing = false
			u.uploadDeadline = time.Now().Add(ttl)
		}
		d.uploadMu.Unlock()
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	blocks := (u.total + nt.ChunkedBinaryBlockSize - 1) / nt.ChunkedBinaryBlockSize
	reservedGrowth := u.total + int64(len(u.key)) + blocks*32 + 512
	releaseMemory, err := d.Engine.ReserveCapacityForMutation(u.key, reservedGrowth)
	if err != nil {
		return fail(err)
	}
	defer releaseMemory()
	f, err := os.Open(u.path)
	if err != nil {
		return fail(err)
	}
	chunked, err := nt.NewChunkedBinaryFromReader(f, u.total)
	_ = f.Close()
	if err != nil {
		return fail(err)
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	v := core.Value{Type: core.TypeBinaryString, Data: chunked}
	if !u.expiresAt.IsZero() {
		err = d.Engine.SetAbsoluteReserved(u.key, v, u.expiresAt, u.nx, u.xx, reservedGrowth)
	} else {
		err = d.Engine.SetReserved(u.key, v, 0, u.nx, u.xx, reservedGrowth)
	}
	if err != nil {
		return fail(err)
	}
	d.uploadMu.Lock()
	if cur := d.uploads[id]; cur == u {
		delete(d.uploads, id)
		d.pendingUploadBytes -= u.total
		if d.pendingUploadBytes < 0 {
			d.pendingUploadBytes = 0
		}
	}
	d.uploadMu.Unlock()
	_ = os.Remove(u.path)
	return true, nil
}
func (d *Dispatcher) blobRead(a args) (any, error) {
	key, err := a.string("key")
	if err != nil {
		return nil, err
	}
	off := a.int64Default("offset", 0)
	n := rawToInt(a.raw("length"), 1<<20)
	if off < 0 || n < 0 || n > maxBlobReadBytes {
		return nil, fmt.Errorf("length must be between 0 and %d", maxBlobReadBytes)
	}
	v, ver, err := d.Engine.Get(key)
	if err != nil {
		return nil, err
	}
	if v.Type != core.TypeBinaryString {
		return nil, core.ErrWrongType
	}
	switch x := v.Data.(type) {
	case *nt.ChunkedBinary:
		b, e := x.ReadRange(off, n)
		if e != nil {
			return nil, e
		}
		return map[string]any{"key": key, "version": ver, "offset": off, "size": x.Size, "data": base64.StdEncoding.EncodeToString(b)}, nil
	case nt.BinaryString:
		if off > int64(len(x)) {
			return nil, core.ErrInvalidArgument
		}
		end := off + int64(n)
		if end > int64(len(x)) {
			end = int64(len(x))
		}
		return map[string]any{"key": key, "version": ver, "offset": off, "size": len(x), "data": base64.StdEncoding.EncodeToString(x[off:end])}, nil
	default:
		return nil, core.ErrWrongType
	}
}

func (d *Dispatcher) blobAbort(ctx context.Context, a args) (any, error) {
	id, err := a.string("upload_id")
	if err != nil {
		return nil, err
	}
	d.uploadMu.Lock()
	u, ok := d.uploads[id]
	if ok && u.owner != "" && uploadOwner(ctx) != "" && u.owner != uploadOwner(ctx) {
		d.uploadMu.Unlock()
		return nil, fmt.Errorf("upload belongs to another connection")
	}
	if ok && u.committing {
		d.uploadMu.Unlock()
		return nil, core.ErrConflict
	}
	if ok {
		d.pendingUploadBytes -= u.total
		if d.pendingUploadBytes < 0 {
			d.pendingUploadBytes = 0
		}
		delete(d.uploads, id)
	}
	d.uploadMu.Unlock()
	if ok {
		_ = os.Remove(u.path)
	}
	return ok, nil
}

func (d *Dispatcher) set(a args) (any, error) {
	k, e := a.string("key")
	if e != nil {
		return nil, e
	}
	tn, e := a.string("type")
	if e != nil {
		return nil, e
	}
	t, e := core.ParseDataType(tn)
	if e != nil {
		return nil, e
	}
	v, e := decodeTypedValue(t, a.raw("value"))
	if e != nil {
		return nil, e
	}
	v, e = d.prepareValue(v)
	if e != nil {
		return nil, e
	}
	if at := a.int64Default("expires_at_unix_nano", 0); at > 0 {
		e = d.Engine.SetAbsolute(k, v, time.Unix(0, at).UTC(), a.bool("nx", false), a.bool("xx", false))
	} else {
		e = d.Engine.Set(k, v, a.duration("ttl_ms", 0), a.bool("nx", false), a.bool("xx", false))
	}
	if e != nil {
		return nil, e
	}
	return true, nil
}
func (d *Dispatcher) create(a args) (any, error) {
	k, e := a.string("key")
	if e != nil {
		return nil, e
	}
	tn, e := a.string("type")
	if e != nil {
		return nil, e
	}
	t, e := core.ParseDataType(tn)
	if e != nil {
		return nil, e
	}
	var optMap map[string]json.RawMessage
	if r := a.raw("options"); len(r) > 0 {
		if e = json.Unmarshal(r, &optMap); e != nil {
			return nil, e
		}
	}
	v, e := createValue(t, args(optMap))
	if e != nil {
		return nil, e
	}
	v, e = d.prepareValue(v)
	if e != nil {
		return nil, e
	}
	if at := a.int64Default("expires_at_unix_nano", 0); at > 0 {
		e = d.Engine.SetAbsolute(k, v, time.Unix(0, at).UTC(), a.bool("nx", false), a.bool("xx", false))
	} else {
		e = d.Engine.Set(k, v, a.duration("ttl_ms", 0), a.bool("nx", false), a.bool("xx", false))
	}
	if e != nil {
		return nil, e
	}
	return true, nil
}
func publicData(x any) any {
	if snap, ok := x.(interface{ Snapshot() any }); ok {
		return snap.Snapshot()
	}
	switch v := x.(type) {
	case nt.BinaryString:
		return base64.StdEncoding.EncodeToString(v)
	case *nt.ChunkedBinary:
		if v.Size > maxBlobReadBytes {
			return map[string]any{"chunked": true, "size": v.Size, "read_command": "BLOB_READ"}
		}
		b, _ := v.ReadRange(0, int(v.Size))
		return base64.StdEncoding.EncodeToString(b)
	case nt.UTF8String:
		return string(v)
	case nt.Integer:
		return int64(v)
	case nt.UnsignedInteger:
		return uint64(v)
	case nt.Float:
		return float64(v)
	case nt.Decimal:
		return v.String()
	case nt.Boolean:
		return bool(v)
	case nt.Null:
		return nil
	case nt.Timestamp:
		return time.Time(v).UTC().Format(time.RFC3339Nano)
	case nt.Duration:
		return time.Duration(v).String()
	case nt.UUID:
		return v.String()
	case nt.BigInteger:
		return v.String()
	case nt.Symbol:
		return string(v)
	case nt.GeoPoint, nt.BoundingBox, nt.Polygon, nt.DenseVector, nt.SparseVector, nt.QuantizedVector,
		nt.FullTextDocument, nt.TagField, nt.NumericField, nt.TextField, nt.TimeSeriesLabels, nt.VectorMetadata:
		return v
	case *nt.Histogram:
		return v.SnapshotAny()
	case *nt.TagField:
		return *v
	case *nt.NumericField:
		return *v
	case *nt.TextField:
		return *v
	case nt.CIDRNetwork:
		return v.CIDR
	case nt.URLValue:
		return v.Raw
	}
	return x
}

func (d *Dispatcher) get(a args) (any, error) {
	k, e := a.string("key")
	if e != nil {
		return nil, e
	}
	v, ver, e := d.Engine.Get(k)
	if e != nil {
		return nil, e
	}
	v, e = d.materializeValue(v)
	if e != nil {
		return nil, e
	}
	ent, _ := d.Engine.GetEntry(k)
	out := map[string]any{"key": k, "type": v.Type.String(), "value": publicData(v.Data), "version": ver}
	if ent != nil && !ent.ExpiresAt.IsZero() {
		out["expires_at"] = ent.ExpiresAt
	}
	return out, nil
}

func argValue(a args, name string) (core.Value, error) {
	r := a.raw(name)
	if len(r) == 0 {
		return core.Value{}, fmt.Errorf("missing %s", name)
	}
	return decodeWireValue(r)
}
func argPoint(a args, name string) (nt.GeoPoint, error) {
	r := a.raw(name)
	var p nt.GeoPoint
	if len(r) == 0 {
		return p, fmt.Errorf("missing %s", name)
	}
	if e := json.Unmarshal(r, &p); e != nil {
		return p, e
	}
	return nt.NewGeoPoint(p.Lat, p.Lon)
}
func argDense(a args, name string) (nt.DenseVector, error) {
	var v []float32
	if e := json.Unmarshal(a.raw(name), &v); e != nil {
		return nil, e
	}
	return nt.DenseVector(v), nil
}
func isReadOp(op string) bool {
	switch op {
	case "GET", "LEN", "RANGE", "VALUES", "MEMBERS", "CONTAINS", "SCORE", "COUNT", "KEYS", "ENTRIES", "PEEK", "STATS", "PENDING", "READ", "GETBIT", "ESTIMATE", "ITEMS", "QUANTILE", "LAST", "AGGREGATE", "SNAPSHOT", "RADIUS", "BOX", "POLYGON", "SEARCH", "LOOKUP", "EXACT", "PREFIX", "NEIGHBORS", "LOAD", "HELD", "REMAINING", "DISTANCE", "DEQUANTIZE", "HOST", "SCHEME", "CARDINALITY", "CMP":
		return true
	}
	return false
}

func mutationMayReduce(op string) bool {
	switch op {
	case "POP", "POP_FRONT", "POP_BACK", "REMOVE", "DELETE", "DEL", "ACK", "NACK", "TRIM", "CLEAR", "DEQUEUE", "RELEASE", "UNLOCK", "LEAVE", "ABORT", "REMOVE_EDGE", "REMOVE_NODE":
		return true
	}
	return false
}
func mutationGrowthEstimate(v core.Value, op string, a args) int64 {
	if mutationMayReduce(op) {
		return 0
	}
	n := int64(64 << 10)
	for _, raw := range a {
		n += int64(len(raw))*16 + 256
	}

	switch v.Type {
	case core.TypeBitmap, core.TypeBitSet:
		if bit, err := a.uint64("bit"); err == nil {
			if x := int64(bit/8 + 1); x > n {
				n = x
			}
		}
	case core.TypeBitField:
		off := a.uint64Default("offset", 0)
		width := a.uint64Default("width", 1)
		if x := int64((off + width + 7) / 8); x > n {
			n = x
		}
	}

	if n > 1<<30 {
		n = 1 << 30
	}
	return n
}

func (d *Dispatcher) op(ctx context.Context, a args) (any, error) {
	k, e := a.string("key")
	if e != nil {
		return nil, e
	}
	op := strings.ToUpper(a.optString("operation", ""))
	if op == "" {
		return nil, fmt.Errorf("operation is required")
	}
	var oa map[string]json.RawMessage
	if r := a.raw("args"); len(r) > 0 {
		if e = json.Unmarshal(r, &oa); e != nil {
			return nil, e
		}
	}
	x := args(oa)
	stored, _, e := d.Engine.Get(k)
	if e != nil {
		return nil, e
	}
	v0, e := d.materializeValue(stored)
	if e != nil {
		return nil, e
	}
	_, wasCompressed := stored.Data.(compression.Payload)
	if isReadOp(op) {
		return d.applyOp(ctx, k, v0, op, x)
	}
	var releaseReservation func()
	reservedGrowth := int64(0)
	if !isReadOp(op) {
		reservedGrowth = mutationGrowthEstimate(v0, op, x)
		releaseReservation, e = d.Engine.ReserveCapacityForMutation(k, reservedGrowth)
		if e != nil {
			return nil, e
		}
		defer releaseReservation()
	}
	if isScalarMutation(v0.Type, op) {
		var result any
		_, e = d.Engine.UpdateReserved(k, core.TypeInvalid, reservedGrowth, func(v core.Value) (core.Value, error) {
			mv, err := d.materializeValue(v)
			if err != nil {
				return core.Value{}, err
			}
			nv, r, err := scalarMutation(mv, op, x)
			if err != nil {
				return core.Value{}, err
			}
			result = r
			return d.prepareValue(nv)
		})
		return result, e
	}
	result, e := d.applyOp(ctx, k, v0, op, x)
	if e != nil {
		return nil, e
	}
	nv, prepErr := d.prepareValue(v0)
	if prepErr != nil {
		return nil, prepErr
	}
	_, nowCompressed := nv.Data.(compression.Payload)
	if wasCompressed || nowCompressed {
		_, err := d.Engine.UpdateReserved(k, v0.Type, reservedGrowth, func(core.Value) (core.Value, error) { return nv, nil })
		if err != nil {
			return nil, err
		}
	} else if e = d.Engine.BumpVersionReserved(k, reservedGrowth); e != nil {
		return nil, e
	}
	return result, nil
}

func (d *Dispatcher) applyOp(ctx context.Context, key string, v core.Value, op string, a args) (any, error) {
	switch x := v.Data.(type) {
	case nt.BinaryString, nt.UTF8String, nt.Symbol, nt.Null, nt.Timestamp, nt.UUID, nt.SparseVector:
		return nil, badOp(v, op)
	case nt.Integer, nt.UnsignedInteger, nt.Float, nt.Boolean, nt.Duration:
		return nil, badOp(v, op)
	case nt.Decimal:
		if op == "CMP" {
			s, e := a.string("value")
			if e != nil {
				return nil, e
			}
			y, e := nt.ParseDecimal(s)
			if e != nil {
				return nil, e
			}
			return x.Cmp(y), nil
		}
		return nil, badOp(v, op)
	case nt.BigInteger:
		return nil, badOp(v, op)
	case nt.GeoPoint:
		if op == "DISTANCE" {
			p, e := argPoint(a, "point")
			if e != nil {
				return nil, e
			}
			return x.DistanceMeters(p), nil
		}
		return nil, badOp(v, op)
	case nt.BoundingBox:
		if op == "CONTAINS" {
			p, e := argPoint(a, "point")
			if e != nil {
				return nil, e
			}
			return x.Contains(p), nil
		}
		return nil, badOp(v, op)
	case nt.Polygon:
		if op == "CONTAINS" {
			p, e := argPoint(a, "point")
			if e != nil {
				return nil, e
			}
			return x.Contains(p), nil
		}
		return nil, badOp(v, op)
	case nt.DenseVector:
		if op == "DISTANCE" {
			q, e := argDense(a, "vector")
			if e != nil {
				return nil, e
			}
			return nt.VectorDistance(x, q, nt.VectorMetric(a.optString("metric", "cosine")))
		}
		return nil, badOp(v, op)
	case nt.QuantizedVector:
		if op == "DEQUANTIZE" {
			return x.Dequantize(), nil
		}
		return nil, badOp(v, op)
	case nt.CIDRNetwork:
		if op == "CONTAINS" {
			ip, e := a.string("ip")
			if e != nil {
				return nil, e
			}
			return x.Contains(ip), nil
		}
		return nil, badOp(v, op)
	case nt.URLValue:
		if op == "HOST" {
			return x.Host(), nil
		}
		if op == "SCHEME" {
			return x.Scheme(), nil
		}
		return nil, badOp(v, op)
	case *nt.List:
		switch op {
		case "PUSH_FRONT":
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			x.PushFront(q)
			return x.Len(), nil
		case "PUSH_BACK":
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			x.PushBack(q)
			return x.Len(), nil
		case "POP_FRONT":
			q, ok := x.PopFront()
			if !ok {
				return nil, nil
			}
			return q, nil
		case "POP_BACK":
			q, ok := x.PopBack()
			if !ok {
				return nil, nil
			}
			return q, nil
		case "RANGE":
			return x.Range(rawToInt(a.raw("start"), 0), rawToInt(a.raw("stop"), -1)), nil
		case "LEN":
			return x.Len(), nil
		}
	case *nt.Array:
		switch op {
		case "APPEND":
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			if e = x.Append(q); e != nil {
				return nil, e
			}
			return x.Len(), nil
		case "SET":
			i, e := a.int("index")
			if e != nil {
				return nil, e
			}
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			return true, x.Set(i, q)
		case "GET":
			i, e := a.int("index")
			if e != nil {
				return nil, e
			}
			q, ok := x.Get(i)
			if !ok {
				return nil, nil
			}
			return q, nil
		case "VALUES":
			return x.Values(), nil
		case "LEN":
			return x.Len(), nil
		}
	case *nt.Deque:
		switch op {
		case "PUSH_FRONT":
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			x.PushFront(q)
			return x.Len(), nil
		case "PUSH_BACK":
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			x.PushBack(q)
			return x.Len(), nil
		case "POP_FRONT":
			q, ok := x.PopFront()
			if !ok {
				return nil, nil
			}
			return q, nil
		case "POP_BACK":
			q, ok := x.PopBack()
			if !ok {
				return nil, nil
			}
			return q, nil
		case "LEN":
			return x.Len(), nil
		case "VALUES":
			return x.Snapshot(), nil
		}
	case *nt.RingBuffer:
		if op == "PUSH" {
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			x.Push(q)
			return x.Len(), nil
		}
		if op == "VALUES" {
			return x.Values(), nil
		}
		if op == "LEN" {
			return x.Len(), nil
		}
	case nt.Tuple:
		if op == "GET" {
			i, e := a.int("index")
			if e != nil {
				return nil, e
			}
			q, ok := x.Get(i)
			if !ok {
				return nil, nil
			}
			return q, nil
		}
		if op == "LEN" {
			return x.Len(), nil
		}
		if op == "VALUES" {
			return x.Items, nil
		}
	case *nt.Set:
		q, e := argValue(a, "value")
		if op != "MEMBERS" && op != "LEN" && e != nil {
			return nil, e
		}
		switch op {
		case "ADD":
			return x.Add(q), nil
		case "REMOVE":
			return x.Remove(q), nil
		case "CONTAINS":
			return x.Contains(q), nil
		case "MEMBERS":
			return x.Values(), nil
		case "LEN":
			return x.Len(), nil
		}
	case *nt.SortedSet:
		switch op {
		case "ADD":
			q, e := argValue(a, "member")
			if e != nil {
				return nil, e
			}
			s, e := a.float("score")
			if e != nil {
				return nil, e
			}
			return x.Add(q, s), nil
		case "REMOVE":
			q, e := argValue(a, "member")
			if e != nil {
				return nil, e
			}
			return x.Remove(q), nil
		case "SCORE":
			q, e := argValue(a, "member")
			if e != nil {
				return nil, e
			}
			s, ok := x.Score(q)
			if !ok {
				return nil, nil
			}
			return s, nil
		case "RANGE":
			return x.Range(rawToInt(a.raw("start"), 0), rawToInt(a.raw("stop"), -1), a.bool("reverse", false)), nil
		}
	case *nt.OrderedSet:
		switch op {
		case "ADD":
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			return x.Add(q), nil
		case "REMOVE":
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			return x.Remove(q), nil
		case "VALUES":
			return x.Values(), nil
		}
	case *nt.MultiSet:
		q, e := argValue(a, "value")
		if e != nil {
			return nil, e
		}
		n := a.uint64Default("count", 1)
		switch op {
		case "ADD":
			return x.Add(q, n), nil
		case "REMOVE":
			return x.Remove(q, n), nil
		case "COUNT":
			return x.Count(q), nil
		}
	case *nt.HashMap:
		if op == "KEYS" {
			return x.Keys(), nil
		}
		f, e := a.string("field")
		if e != nil {
			return nil, e
		}
		switch op {
		case "SET":
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			x.Set(f, q)
			return true, nil
		case "GET":
			q, ok := x.Get(f)
			if !ok {
				return nil, nil
			}
			return q, nil
		case "DELETE":
			return x.Delete(f), nil
		}
	case *nt.OrderedMap:
		if op == "ENTRIES" {
			return x.Entries(), nil
		}
		f, e := a.string("field")
		if e != nil {
			return nil, e
		}
		switch op {
		case "SET":
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			x.Set(f, q)
			return true, nil
		case "GET":
			q, ok := x.Get(f)
			if !ok {
				return nil, nil
			}
			return q, nil
		case "DELETE":
			return x.Delete(f), nil
		}
	case *nt.MultiMap:
		f, e := a.string("field")
		if e != nil {
			return nil, e
		}
		switch op {
		case "ADD":
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			x.Add(f, q)
			return true, nil
		case "GET":
			return x.Get(f), nil
		case "REMOVE":
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			return x.RemoveValue(f, q), nil
		}
	case *nt.JSONDocument:
		if op == "GET" {
			return x.Value(), nil
		}
		if op == "SET" {
			var z any
			if e := json.Unmarshal(a.raw("value"), &z); e != nil {
				return nil, e
			}
			return true, x.Set(z)
		}
	case *nt.Record:
		f, e := a.string("field")
		if e != nil {
			return nil, e
		}
		if op == "SET" {
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			return true, x.Set(f, q)
		}
		if op == "GET" {
			q, ok := x.Get(f)
			if !ok {
				return nil, nil
			}
			return q, nil
		}
	case *nt.Tensor:
		var idx []int
		_ = json.Unmarshal(a.raw("indices"), &idx)
		if op == "GET" {
			return x.Get(idx...)
		}
		if op == "SET" {
			n, e := a.float("value")
			if e != nil {
				return nil, e
			}
			return true, x.Set(n, idx...)
		}
		if op == "LEN" {
			return x.Size(), nil
		}
	case *nt.FIFOQueue:
		if op == "ENQUEUE" {
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			x.Enqueue(q)
			return x.Len(), nil
		}
		if op == "DEQUEUE" {
			q, ok := x.Dequeue()
			if !ok {
				return nil, nil
			}
			return q, nil
		}
		if op == "PEEK" {
			q, ok := x.Peek()
			if !ok {
				return nil, nil
			}
			return q, nil
		}
		if op == "LEN" {
			return x.Len(), nil
		}
	case *nt.LIFOStack:
		if op == "PUSH" {
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			x.Push(q)
			return x.Len(), nil
		}
		if op == "POP" {
			q, ok := x.Pop()
			if !ok {
				return nil, nil
			}
			return q, nil
		}
		if op == "PEEK" {
			q, ok := x.Peek()
			if !ok {
				return nil, nil
			}
			return q, nil
		}
		if op == "LEN" {
			return x.Len(), nil
		}
	case *nt.PriorityQueue:
		if op == "ENQUEUE" {
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			p, e := a.float("priority")
			if e != nil {
				return nil, e
			}
			x.Enqueue(q, p)
			return x.Len(), nil
		}
		if op == "DEQUEUE" {
			q, ok := x.Dequeue()
			if !ok {
				return nil, nil
			}
			return q, nil
		}
		if op == "LEN" {
			return x.Len(), nil
		}
	case *nt.BlockingQueue:
		if op == "ENQUEUE" {
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			return true, x.Enqueue(q)
		}
		if op == "DEQUEUE" {
			return x.Dequeue(ctx)
		}
		if op == "LEN" {
			return x.Len(), nil
		}
		if op == "CLOSE" {
			x.Close()
			return true, nil
		}
	case *nt.DelayedQueue:
		if op == "ENQUEUE" {
			id, e := a.string("id")
			if e != nil {
				return nil, e
			}
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			atMs, e := a.int64("available_at_ms")
			if e != nil {
				return nil, e
			}
			x.Enqueue(id, q, time.UnixMilli(atMs))
			return x.Len(), nil
		}
		if op == "POP_READY" {
			now := time.Now()
			if n := a.int64Default("now_unix_nano", 0); n > 0 {
				now = time.Unix(0, n)
			}
			q, ok := x.PopReady(now)
			if !ok {
				return nil, nil
			}
			return q, nil
		}
		if op == "LEN" {
			return x.Len(), nil
		}
	case *nt.ReliableQueue:
		if op == "ENQUEUE" {
			id, e := a.string("id")
			if e != nil {
				return nil, e
			}
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			now := time.Now().UTC()
			if n := a.int64Default("now_unix_nano", 0); n > 0 {
				now = time.Unix(0, n).UTC()
			}
			x.EnqueueAt(id, q, now)
			return true, nil
		}
		if op == "CLAIM" {
			now := time.Now()
			if n := a.int64Default("now_unix_nano", 0); n > 0 {
				now = time.Unix(0, n)
			}
			q, ok := x.ClaimAt(a.duration("lease_ms", 30*time.Second), now)
			if !ok {
				return nil, nil
			}
			return q, nil
		}
		if op == "ACK" {
			id, e := a.string("id")
			if e != nil {
				return nil, e
			}
			return x.Ack(id), nil
		}
		if op == "NACK" {
			id, e := a.string("id")
			if e != nil {
				return nil, e
			}
			return x.Nack(id), nil
		}
		if op == "STATS" {
			r, p, dead := x.Stats()
			return map[string]int{"ready": r, "pending": p, "dead": dead}, nil
		}
	case *nt.Stream:
		return d.streamOp(x, op, a)
	case *nt.PubSub:
		if op == "STATS" {
			return x.Stats(), nil
		}
		if op == "PUBLISH" {
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			ch, e := a.string("channel")
			if e != nil {
				return nil, e
			}
			return x.Publish(ch, q), nil
		}
		return nil, badOp(v, op)
	case *nt.ConsumerGroup:
		if op == "SNAPSHOT" {
			return x.Snapshot(), nil
		}
		return nil, badOp(v, op)
	case *nt.PersistentTopic:
		if op == "PUBLISH" {
			q, e := argValue(a, "value")
			if e != nil {
				return nil, e
			}
			at := time.Now().UTC()
			if n := a.int64Default("timestamp_unix_nano", 0); n > 0 {
				at = time.Unix(0, n).UTC()
			}
			return x.PublishAt(a.optString("message_key", ""), q, at), nil
		}
		if op == "READ" {
			from, e := a.uint64("from")
			if e != nil {
				return nil, e
			}
			return x.Read(from, rawToInt(a.raw("count"), 100)), nil
		}
	case *nt.EventLog:
		if op == "APPEND" {
			typ, e := a.string("event_type")
			if e != nil {
				return nil, e
			}
			q, e := argValue(a, "payload")
			if e != nil {
				return nil, e
			}
			at := time.Now().UTC()
			if n := a.int64Default("timestamp_unix_nano", 0); n > 0 {
				at = time.Unix(0, n).UTC()
			}
			return x.AppendAt(typ, a.optString("aggregate_id", ""), q, at), nil
		}
		if op == "READ" {
			return x.Read(a.uint64Default("from", 1), rawToInt(a.raw("count"), 100)), nil
		}
		if op == "LEN" {
			return x.Len(), nil
		}
	case *nt.Bitmap:
		bit, e := a.uint64("bit")
		if op != "COUNT" && e != nil {
			return nil, e
		}
		if op == "SET" {
			return x.Set(bit, a.bool("value", true)), nil
		}
		if op == "GETBIT" {
			return x.Get(bit), nil
		}
		if op == "COUNT" {
			return x.Count(), nil
		}
	case *nt.BitField:
		off, e := a.uint64("offset")
		if e != nil {
			return nil, e
		}
		w, e := a.uint64("width")
		if e != nil {
			return nil, e
		}
		if op == "SET" {
			n, e := a.uint64("value")
			if e != nil {
				return nil, e
			}
			return true, x.SetUnsigned(off, w, n)
		}
		if op == "GET" {
			return x.GetUnsigned(off, w)
		}
	case *nt.RoaringBitmap:
		n, e := a.uint64("value")
		if op != "VALUES" && op != "CARDINALITY" && e != nil {
			return nil, e
		}
		if n > 1<<32-1 {
			return nil, fmt.Errorf("value out of uint32 range")
		}
		switch op {
		case "ADD":
			return x.Add(uint32(n)), nil
		case "REMOVE":
			return x.Remove(uint32(n)), nil
		case "CONTAINS":
			return x.Contains(uint32(n)), nil
		case "VALUES":
			return x.Values(), nil
		case "CARDINALITY":
			return x.Cardinality(), nil
		}
	case *nt.HyperLogLog:
		if op == "ADD" {
			s, e := a.string("value")
			if e != nil {
				return nil, e
			}
			return x.Add([]byte(s)), nil
		}
		if op == "COUNT" {
			return x.Count(), nil
		}
	case *nt.BloomFilter:
		s, e := a.string("value")
		if e != nil {
			return nil, e
		}
		if op == "ADD" {
			return x.Add([]byte(s)), nil
		}
		if op == "CONTAINS" {
			return x.Contains([]byte(s)), nil
		}
	case *nt.CountingBloomFilter:
		s, e := a.string("value")
		if e != nil {
			return nil, e
		}
		if op == "ADD" {
			x.Add([]byte(s))
			return true, nil
		}
		if op == "REMOVE" {
			return x.Remove([]byte(s)), nil
		}
		if op == "CONTAINS" {
			return x.Contains([]byte(s)), nil
		}
	case *nt.CuckooFilter:
		s, e := a.string("value")
		if e != nil {
			return nil, e
		}
		if op == "ADD" {
			return x.Add([]byte(s)), nil
		}
		if op == "REMOVE" {
			return x.Remove([]byte(s)), nil
		}
		if op == "CONTAINS" {
			return x.Contains([]byte(s)), nil
		}
	case *nt.CountMinSketch:
		s, e := a.string("value")
		if e != nil {
			return nil, e
		}
		if op == "ADD" {
			return x.Add([]byte(s), a.uint64Default("count", 1)), nil
		}
		if op == "ESTIMATE" {
			return x.Estimate([]byte(s)), nil
		}
	case *nt.TopK:
		s, e := a.string("value")
		if op != "ITEMS" && e != nil {
			return nil, e
		}
		if op == "ADD" {
			return x.Add(s, a.uint64Default("count", 1)), nil
		}
		if op == "ITEMS" {
			return x.Items(), nil
		}
	case *nt.TDigest:
		if op == "ADD" {
			n, e := a.float("value")
			if e != nil {
				return nil, e
			}
			x.Add(n, optFloat(a, "weight", 1))
			return true, nil
		}
		if op == "QUANTILE" {
			q, e := a.float("q")
			if e != nil {
				return nil, e
			}
			return x.Quantile(q), nil
		}
	case *nt.TimeSeries:
		return timeSeriesOp(x, op, a)
	case *nt.CounterSeries:
		if op == "ADD" {
			ts := a.int64Default("timestamp", time.Now().UnixMilli())
			n, e := a.float("value")
			if e != nil {
				return nil, e
			}
			return true, x.Add(ts, n)
		}
		return timeSeriesOp(x.Series, op, a)
	case *nt.GaugeSeries:
		if op == "SET" {
			ts := a.int64Default("timestamp", time.Now().UnixMilli())
			n, e := a.float("value")
			if e != nil {
				return nil, e
			}
			return true, x.Set(ts, n)
		}
		return timeSeriesOp(x.Series, op, a)
	case *nt.Histogram:
		if op == "OBSERVE" {
			n, e := a.float("value")
			if e != nil {
				return nil, e
			}
			x.Observe(n)
			return true, nil
		}
		if op == "SNAPSHOT" {
			return x.SnapshotAny(), nil
		}
	case *nt.AggregatedSeries:
		if op == "RANGE" || op == "AGGREGATE" {
			sourceKey, e := a.string("source_key")
			if e != nil {
				return nil, e
			}
			sv, _, e := d.Engine.Get(sourceKey)
			if e != nil {
				return nil, e
			}
			var source *nt.TimeSeries
			switch z := sv.Data.(type) {
			case *nt.TimeSeries:
				source = z
			case *nt.CounterSeries:
				source = z.Series
			case *nt.GaugeSeries:
				source = z.Series
			default:
				return nil, core.ErrWrongType
			}
			from := a.int64Default("from", 0)
			to := a.int64Default("to", time.Now().UnixMilli())
			bucket := x.BucketMillis
			if b := a.int64Default("bucket_ms", 0); b > 0 {
				bucket = b
			}
			agg := x.Aggregation
			if av := a.optString("aggregation", ""); av != "" {
				agg = nt.Aggregation(av)
			}
			return source.Aggregate(from, to, bucket, agg)
		}
		return nil, badOp(v, op)
	case nt.TimeSeriesLabels:
		if op == "GET" {
			return map[string]string(x), nil
		}
		if op == "SET" {
			k, e := a.string("label")
			if e != nil {
				return nil, e
			}
			val, e := a.string("value")
			if e != nil {
				return nil, e
			}
			x[k] = val
			return true, nil
		}
	case *nt.GeoSpatialIndex:
		return geoOp(x, op, a)
	case *nt.VectorSet:
		return vectorSetOp(x, op, a)
	case *nt.FlatVectorIndex:
		return vectorSetOp(x.Set, op, a)
	case *nt.HNSWIndex:
		return hnswOp(x, op, a)
	case *nt.BTreeIndex:
		return btreeOp(x, op, a)
	case *nt.HashIndex:
		val, e := a.string("value")
		if e != nil {
			return nil, e
		}
		if op == "ADD" {
			id, e := a.string("id")
			if e != nil {
				return nil, e
			}
			x.Add(val, id)
			return true, nil
		}
		if op == "REMOVE" {
			id, e := a.string("id")
			if e != nil {
				return nil, e
			}
			return x.Remove(val, id), nil
		}
		if op == "LOOKUP" {
			return x.Lookup(val), nil
		}
	case *nt.RadixTree:
		if op == "INSERT" {
			s, e := a.string("value")
			if e != nil {
				return nil, e
			}
			id, e := a.string("id")
			if e != nil {
				return nil, e
			}
			x.Insert(s, id)
			return true, nil
		}
		if op == "EXACT" {
			s, e := a.string("value")
			if e != nil {
				return nil, e
			}
			return x.Exact(s), nil
		}
		if op == "PREFIX" {
			s, e := a.string("prefix")
			if e != nil {
				return nil, e
			}
			return x.Prefix(s, rawToInt(a.raw("limit"), 100)), nil
		}
	case *nt.InvertedIndex:
		if op == "INDEX" {
			id, e := a.string("id")
			if e != nil {
				return nil, e
			}
			text, e := a.string("text")
			if e != nil {
				return nil, e
			}
			x.Index(id, text)
			return true, nil
		}
		if op == "REMOVE" {
			id, e := a.string("id")
			if e != nil {
				return nil, e
			}
			x.Remove(id)
			return true, nil
		}
		if op == "SEARCH" {
			q, e := a.string("query")
			if e != nil {
				return nil, e
			}
			return x.Search(q, rawToInt(a.raw("limit"), 20)), nil
		}
	case *nt.SecondaryIndex:
		field, e := a.string("field")
		if e != nil {
			return nil, e
		}
		val, e := a.string("value")
		if e != nil {
			return nil, e
		}
		if op == "ADD" {
			id, e := a.string("id")
			if e != nil {
				return nil, e
			}
			x.Add(field, val, id)
			return true, nil
		}
		if op == "LOOKUP" {
			return x.Lookup(field, val), nil
		}
	case nt.VectorMetadata:
		if op == "GET" {
			out := map[string]any{}
			for k, v := range x {
				out[k] = v
			}
			return out, nil
		}
		if op == "SET" {
			field, e := a.string("field")
			if e != nil {
				return nil, e
			}
			var value any
			if e = json.Unmarshal(a.raw("value"), &value); e != nil {
				return nil, e
			}
			x[field] = value
			return true, nil
		}
		if op == "DELETE" {
			field, e := a.string("field")
			if e != nil {
				return nil, e
			}
			_, ok := x[field]
			delete(x, field)
			return ok, nil
		}
	case *nt.FullTextDocument:
		switch op {
		case "ADD_TEXT":
			name, e := a.string("name")
			if e != nil {
				return nil, e
			}
			value, e := a.string("value")
			if e != nil {
				return nil, e
			}
			x.AddText(nt.TextField{Name: name, Value: value, Weight: optFloat(a, "weight", 1)})
			return true, nil
		case "ADD_TAG":
			name, e := a.string("name")
			if e != nil {
				return nil, e
			}
			var vals []string
			if e = json.Unmarshal(a.raw("values"), &vals); e != nil {
				return nil, e
			}
			x.AddTag(nt.TagField{Name: name, Values: vals})
			return true, nil
		case "ADD_NUMERIC":
			name, e := a.string("name")
			if e != nil {
				return nil, e
			}
			value, e := a.float("value")
			if e != nil {
				return nil, e
			}
			x.AddNumeric(nt.NumericField{Name: name, Value: value})
			return true, nil
		case "GET", "SNAPSHOT":
			return x.Snapshot(), nil
		}
	case *nt.TagField:
		if op == "SET" {
			x.Name = a.optString("name", x.Name)
			var vals []string
			if r := a.raw("values"); len(r) > 0 {
				if e := json.Unmarshal(r, &vals); e != nil {
					return nil, e
				}
				x.Values = vals
			}
			return true, nil
		}
		if op == "GET" {
			return *x, nil
		}
	case *nt.NumericField:
		if op == "SET" {
			x.Name = a.optString("name", x.Name)
			if r := a.raw("value"); len(r) > 0 {
				n, e := a.float("value")
				if e != nil {
					return nil, e
				}
				x.Value = n
			}
			return true, nil
		}
		if op == "GET" {
			return *x, nil
		}
	case *nt.TextField:
		if op == "SET" {
			x.Name = a.optString("name", x.Name)
			if r := a.raw("value"); len(r) > 0 {
				sv, e := a.string("value")
				if e != nil {
					return nil, e
				}
				x.Value = sv
			}
			if r := a.raw("weight"); len(r) > 0 {
				x.Weight = optFloat(a, "weight", x.Weight)
			}
			return true, nil
		}
		if op == "GET" {
			return *x, nil
		}
	case *nt.AdjacencySet:
		from, e := a.string("from")
		if e != nil {
			return nil, e
		}
		if op == "NEIGHBORS" {
			return x.Neighbors(from), nil
		}
		to, e := a.string("to")
		if e != nil {
			return nil, e
		}
		if op == "ADD" {
			x.Add(from, to, optFloat(a, "weight", 1))
			return true, nil
		}
		if op == "REMOVE" {
			return x.Remove(from, to), nil
		}
	case *nt.AtomicCounter:
		if op == "ADD" {
			n, e := a.int64("delta")
			if e != nil {
				return nil, e
			}
			return x.Add(n), nil
		}
		if op == "LOAD" {
			return x.Load(), nil
		}
		if op == "CAS" {
			old, e := a.int64("old")
			if e != nil {
				return nil, e
			}
			n, e := a.int64("new")
			if e != nil {
				return nil, e
			}
			return x.CompareAndSwap(old, n), nil
		}
	case *nt.Lease:
		return leaseOp(x, op, a)
	case *nt.Lock:
		if op == "TRY_LOCK" {
			owner, e := a.string("owner")
			if e != nil {
				return nil, e
			}
			token, e := a.string("token")
			if e != nil {
				return nil, e
			}
			return x.TryLock(owner, token, a.duration("ttl_ms", 30*time.Second)), nil
		}
		if op == "UNLOCK" {
			token, e := a.string("token")
			if e != nil {
				return nil, e
			}
			return x.Unlock(token), nil
		}
		if op == "RENEW" {
			token, e := a.string("token")
			if e != nil {
				return nil, e
			}
			return x.Renew(token, a.duration("ttl_ms", 30*time.Second)), nil
		}
	case *nt.Barrier:
		if op == "WAIT" {
			return true, x.Wait(ctx)
		}
	case *nt.Semaphore:
		if op == "ACQUIRE" {
			return true, x.Acquire(ctx)
		}
		if op == "TRY_ACQUIRE" {
			return x.TryAcquire(), nil
		}
		if op == "RELEASE" {
			return x.Release(), nil
		}
		if op == "STATS" {
			return x.Snapshot(), nil
		}
	case *nt.RateLimitBucket:
		if op == "ALLOW" {
			return x.Allow(optFloat(a, "cost", 1), time.Now()), nil
		}
		if op == "REMAINING" {
			return x.Remaining(time.Now()), nil
		}
	}
	return nil, badOp(v, op)
}

func isScalarMutation(t core.DataType, op string) bool {
	switch t {
	case core.TypeInteger, core.TypeUnsignedInteger, core.TypeFloat, core.TypeBigInteger:
		return op == "ADD"
	case core.TypeDecimal, core.TypeBigDecimal:
		return op == "ADD" || op == "SUB"
	case core.TypeBoolean:
		return op == "TOGGLE"
	case core.TypeDuration:
		return op == "ADD"
	}
	return false
}

func scalarMutation(v core.Value, op string, a args) (core.Value, any, error) {
	switch x := v.Data.(type) {
	case nt.Integer:
		d, e := a.int64("delta")
		if e != nil {
			return v, nil, e
		}
		n := nt.Integer(int64(x) + d)
		v.Data = n
		return v, int64(n), nil
	case nt.UnsignedInteger:
		d, e := a.uint64("delta")
		if e != nil {
			return v, nil, e
		}
		n := nt.UnsignedInteger(uint64(x) + d)
		v.Data = n
		return v, uint64(n), nil
	case nt.Float:
		d, e := a.float("delta")
		if e != nil {
			return v, nil, e
		}
		n := nt.Float(float64(x) + d)
		v.Data = n
		return v, float64(n), nil
	case nt.Decimal:
		s, e := a.string("value")
		if e != nil {
			return v, nil, e
		}
		y, e := nt.ParseDecimal(s)
		if e != nil {
			return v, nil, e
		}
		n := x.Add(y)
		if op == "SUB" {
			n = x.Sub(y)
		}
		v.Data = n
		return v, n.String(), nil
	case nt.BigInteger:
		s, e := a.string("value")
		if e != nil {
			return v, nil, e
		}
		y := new(big.Int)
		if _, ok := y.SetString(s, 10); !ok {
			return v, nil, fmt.Errorf("invalid big integer")
		}
		z := new(big.Int).Add(&x.Int, y)
		n := nt.BigInteger{}
		n.Int.Set(z)
		v.Data = n
		return v, n.String(), nil
	case nt.Boolean:
		n := nt.Boolean(!bool(x))
		v.Data = n
		return v, bool(n), nil
	case nt.Duration:
		d := a.duration("delta_ms", 0)
		n := nt.Duration(time.Duration(x) + d)
		v.Data = n
		return v, time.Duration(n).String(), nil
	default:
		return v, nil, badOp(v, op)
	}
}

func badOp(v core.Value, op string) error {
	return fmt.Errorf("operation %s is not supported by %s", op, v.Type)
}

func (a args) uint64Default(name string, def uint64) uint64 {
	n, e := a.uint64(name)
	if e != nil {
		return def
	}
	return n
}
func (a args) int64Default(name string, def int64) int64 {
	n, e := a.int64(name)
	if e != nil {
		return def
	}
	return n
}

func (d *Dispatcher) streamOp(s *nt.Stream, op string, a args) (any, error) {
	switch op {
	case "ADD":
		var fieldsRaw map[string]json.RawMessage
		if e := json.Unmarshal(a.raw("fields"), &fieldsRaw); e != nil {
			return nil, e
		}
		fields := map[string]core.Value{}
		for k, r := range fieldsRaw {
			v, e := decodeWireValue(r)
			if e != nil {
				return nil, fmt.Errorf("field %s: %w", k, e)
			}
			fields[k] = v
		}
		var id *nt.StreamID
		if raw := a.optString("id", ""); raw != "" {
			parsed, e := nt.ParseStreamID(raw)
			if e != nil {
				return nil, e
			}
			id = &parsed
		}
		return s.Add(fields, id)
	case "RANGE":
		return s.Range(nil, nil, rawToInt(a.raw("count"), 100)), nil
	case "GROUP_CREATE":
		name, e := a.string("group")
		if e != nil {
			return nil, e
		}
		return true, s.CreateGroup(name, nt.StreamID{})
	case "GROUP_READ":
		g, e := a.string("group")
		if e != nil {
			return nil, e
		}
		c, e := a.string("consumer")
		if e != nil {
			return nil, e
		}
		return s.ReadGroup(g, c, rawToInt(a.raw("count"), 10))
	case "ACK":
		g, e := a.string("group")
		if e != nil {
			return nil, e
		}
		ss, e := a.strings("ids")
		if e != nil {
			return nil, e
		}
		ids := make([]nt.StreamID, 0, len(ss))
		for _, z := range ss {
			id, e := nt.ParseStreamID(z)
			if e != nil {
				return nil, e
			}
			ids = append(ids, id)
		}
		return s.Ack(g, ids...)
	case "PENDING":
		g, e := a.string("group")
		if e != nil {
			return nil, e
		}
		return s.Pending(g)
	case "LEN":
		return s.Len(), nil
	}
	return nil, fmt.Errorf("unknown stream operation %s", op)
}
func timeSeriesOp(s *nt.TimeSeries, op string, a args) (any, error) {
	switch op {
	case "ADD":
		ts := a.int64Default("timestamp", time.Now().UnixMilli())
		n, e := a.float("value")
		if e != nil {
			return nil, e
		}
		return true, s.Add(ts, n)
	case "RANGE":
		from := a.int64Default("from", 0)
		to := a.int64Default("to", time.Now().UnixMilli())
		return s.Range(from, to), nil
	case "LAST":
		p, ok := s.Last()
		if !ok {
			return nil, nil
		}
		return p, nil
	case "AGGREGATE":
		return s.Aggregate(a.int64Default("from", 0), a.int64Default("to", time.Now().UnixMilli()), a.int64Default("bucket_ms", 60000), nt.Aggregation(a.optString("aggregation", "avg")))
	case "LEN":
		return s.Len(), nil
	}
	return nil, fmt.Errorf("unknown timeseries operation %s", op)
}
func geoOp(g *nt.GeoSpatialIndex, op string, a args) (any, error) {
	switch op {
	case "ADD":
		id, e := a.string("id")
		if e != nil {
			return nil, e
		}
		p, e := argPoint(a, "point")
		if e != nil {
			return nil, e
		}
		var m map[string]string
		_ = json.Unmarshal(a.raw("metadata"), &m)
		g.Add(id, p, m)
		return true, nil
	case "REMOVE":
		id, e := a.string("id")
		if e != nil {
			return nil, e
		}
		return g.Remove(id), nil
	case "RADIUS":
		p, e := argPoint(a, "center")
		if e != nil {
			return nil, e
		}
		r, e := a.float("radius_meters")
		if e != nil {
			return nil, e
		}
		return g.WithinRadius(p, r, rawToInt(a.raw("limit"), 100)), nil
	case "BOX":
		var b nt.BoundingBox
		if e := json.Unmarshal(a.raw("box"), &b); e != nil {
			return nil, e
		}
		return g.WithinBox(b), nil
	case "POLYGON":
		var pts []nt.GeoPoint
		if e := json.Unmarshal(a.raw("points"), &pts); e != nil {
			return nil, e
		}
		p, e := nt.NewPolygon(pts)
		if e != nil {
			return nil, e
		}
		return g.WithinPolygon(p), nil
	}
	return nil, fmt.Errorf("unknown geo operation %s", op)
}
func vectorSetOp(s *nt.VectorSet, op string, a args) (any, error) {
	switch op {
	case "ADD":
		id, e := a.string("id")
		if e != nil {
			return nil, e
		}
		v, e := argDense(a, "vector")
		if e != nil {
			return nil, e
		}
		var m nt.VectorMetadata
		_ = json.Unmarshal(a.raw("metadata"), &m)
		return true, s.Add(id, v, m)
	case "REMOVE":
		id, e := a.string("id")
		if e != nil {
			return nil, e
		}
		return s.Remove(id), nil
	case "SEARCH":
		q, e := argDense(a, "vector")
		if e != nil {
			return nil, e
		}
		return s.Search(q, rawToInt(a.raw("k"), 10), nil)
	case "LEN":
		return s.Len(), nil
	}
	return nil, fmt.Errorf("unknown vector operation %s", op)
}
func hnswOp(h *nt.HNSWIndex, op string, a args) (any, error) {
	switch op {
	case "ADD":
		id, e := a.string("id")
		if e != nil {
			return nil, e
		}
		v, e := argDense(a, "vector")
		if e != nil {
			return nil, e
		}
		var m nt.VectorMetadata
		_ = json.Unmarshal(a.raw("metadata"), &m)
		return true, h.Add(id, v, m)
	case "REMOVE":
		id, e := a.string("id")
		if e != nil {
			return nil, e
		}
		return h.Remove(id), nil
	case "SEARCH":
		q, e := argDense(a, "vector")
		if e != nil {
			return nil, e
		}
		return h.Search(q, rawToInt(a.raw("k"), 10), nil)
	}
	return nil, fmt.Errorf("unknown hnsw operation %s", op)
}
func btreeOp(b *nt.BTreeIndex, op string, a args) (any, error) {
	value, e := a.string("value")
	if e != nil {
		return nil, e
	}
	switch op {
	case "INSERT":
		id, e := a.string("id")
		if e != nil {
			return nil, e
		}
		b.Insert(value, id)
		return true, nil
	case "DELETE":
		id, e := a.string("id")
		if e != nil {
			return nil, e
		}
		return b.Delete(value, id), nil
	case "SEARCH":
		return b.Search(value), nil
	case "RANGE":
		return b.Range(value, a.optString("max", ""), rawToInt(a.raw("limit"), 100)), nil
	}
	return nil, fmt.Errorf("unknown btree operation %s", op)
}
func leaseOp(l *nt.Lease, op string, a args) (any, error) {
	now := time.Now()
	switch op {
	case "ACQUIRE":
		owner, e := a.string("owner")
		if e != nil {
			return nil, e
		}
		token, e := a.string("token")
		if e != nil {
			return nil, e
		}
		return l.Acquire(owner, token, a.duration("ttl_ms", 30*time.Second), now), nil
	case "RENEW":
		token, e := a.string("token")
		if e != nil {
			return nil, e
		}
		return l.Renew(token, a.duration("ttl_ms", 30*time.Second), now), nil
	case "RELEASE":
		token, e := a.string("token")
		if e != nil {
			return nil, e
		}
		return l.Release(token), nil
	case "HELD":
		return l.Held(now), nil
	}
	return nil, fmt.Errorf("unknown lease operation %s", op)
}

var _ = sort.Strings
var _ = errors.Is
