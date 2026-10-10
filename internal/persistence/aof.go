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
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/newfoundcodes/amaquet/internal/protocol"
)

const (
	aofHeaderV1       = "AMQTAOF1\n"
	aofHeaderV2       = "AMQTAOF2\n"
	preRenameHeaderV1 = "\x4e\x46\x44\x42AOF1\n"
	preRenameHeaderV2 = "\x4e\x46\x44\x42AOF2\n"
	maxRecord         = 64 << 20
)

type Record struct {
	Kind               string            `json:"kind"`
	TxID               uint64            `json:"tx_id"`
	Request            *protocol.Request `json:"request,omitempty"`
	RecordedAtUnixNano int64             `json:"recorded_at_unix_nano,omitempty"`
}

type AOF struct {
	mu        sync.Mutex
	file      *os.File
	writer    *bufio.Writer
	fsync     string
	stop      chan struct{}
	done      chan struct{}
	next      atomic.Uint64
	healthMu  sync.RWMutex
	healthErr error
}

func Open(path, fsync string) (*AOF, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("AOF path is required")
	}
	if fsync == "" {
		fsync = "everysec"
	}
	if fsync != "always" && fsync != "everysec" && fsync != "no" {
		return nil, fmt.Errorf("invalid AOF fsync mode %q", fsync)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	if err := migratePreRenameV2(path); err != nil {
		return nil, err
	}
	if err := migrateV1(path); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if st.Size() == 0 {
		if _, err = f.WriteString(aofHeaderV2); err != nil {
			f.Close()
			return nil, err
		}
		if err = f.Sync(); err != nil {
			f.Close()
			return nil, err
		}
	} else {
		h := make([]byte, len(aofHeaderV2))
		if _, err = f.ReadAt(h, 0); err != nil || string(h) != aofHeaderV2 {
			f.Close()
			return nil, fmt.Errorf("invalid Amaquet AOF header")
		}
	}
	if _, err = f.Seek(0, io.SeekEnd); err != nil {
		f.Close()
		return nil, err
	}
	a := &AOF{file: f, writer: bufio.NewWriterSize(f, 64<<10), fsync: fsync, stop: make(chan struct{}), done: make(chan struct{})}
	maxTx, _, _ := scanV2(path, nil)
	a.next.Store(maxTx)
	if fsync == "everysec" {
		go a.syncLoop()
	} else {
		close(a.done)
	}
	return a, nil
}

func (a *AOF) setHealth(err error) {
	if err == nil {
		return
	}
	a.healthMu.Lock()
	if a.healthErr == nil {
		a.healthErr = err
	}
	a.healthMu.Unlock()
}
func (a *AOF) HealthError() error { a.healthMu.RLock(); defer a.healthMu.RUnlock(); return a.healthErr }
func (a *AOF) appendRecordLocked(rec Record) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if len(b) > maxRecord {
		return fmt.Errorf("AOF record too large")
	}
	var h [8]byte
	binary.BigEndian.PutUint32(h[0:4], uint32(len(b)))
	binary.BigEndian.PutUint32(h[4:8], crc32.ChecksumIEEE(b))
	if _, err = a.writer.Write(h[:]); err != nil {
		return err
	}
	if _, err = a.writer.Write(b); err != nil {
		return err
	}
	if a.fsync == "always" {
		if err = a.writer.Flush(); err != nil {
			return err
		}
		return a.file.Sync()
	}
	return nil
}
func (a *AOF) Prepare(req protocol.Request) (uint64, error) {
	if err := a.HealthError(); err != nil {
		return 0, fmt.Errorf("AOF unhealthy: %w", err)
	}
	id := a.next.Add(1)
	cp := req
	rec := Record{Kind: "prepare", TxID: id, Request: &cp, RecordedAtUnixNano: time.Now().UnixNano()}
	a.mu.Lock()
	err := a.appendRecordLocked(rec)
	a.mu.Unlock()
	if err != nil {
		a.setHealth(err)
		return 0, err
	}
	return id, nil
}
func (a *AOF) Commit(id uint64) error {
	a.mu.Lock()
	err := a.appendRecordLocked(Record{Kind: "commit", TxID: id})
	a.mu.Unlock()
	if err != nil {
		a.setHealth(err)
	}
	return err
}
func (a *AOF) Abort(id uint64) error {
	a.mu.Lock()
	err := a.appendRecordLocked(Record{Kind: "abort", TxID: id})
	a.mu.Unlock()
	if err != nil {
		a.setHealth(err)
	}
	return err
}

func (a *AOF) Append(req protocol.Request) error {
	id, err := a.Prepare(req)
	if err != nil {
		return err
	}
	return a.Commit(id)
}
func (a *AOF) flushLocked() error {
	if err := a.writer.Flush(); err != nil {
		a.setHealth(err)
		return err
	}
	if a.fsync != "no" {
		if err := a.file.Sync(); err != nil {
			a.setHealth(err)
			return err
		}
	}
	return nil
}
func (a *AOF) Flush() error { a.mu.Lock(); defer a.mu.Unlock(); return a.flushLocked() }
func (a *AOF) syncLoop() {
	defer close(a.done)
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			if err := a.Flush(); err != nil {
				a.setHealth(err)
			}
		case <-a.stop:
			_ = a.Flush()
			return
		}
	}
}
func (a *AOF) Close() error {
	if a.fsync == "everysec" {
		select {
		case <-a.stop:
		default:
			close(a.stop)
		}
		<-a.done
	}
	if err := a.Flush(); err != nil {
		_ = a.file.Close()
		return err
	}
	return a.file.Close()
}

func Replay(path string, apply func(protocol.Request) error) (int, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer f.Close()
	h := make([]byte, len(aofHeaderV2))
	n, err := io.ReadFull(f, h)
	if err != nil {
		return 0, err
	}
	_ = n
	header := string(h)
	if header == aofHeaderV1 || header == preRenameHeaderV1 {
		if _, err = f.Seek(int64(len(aofHeaderV1)), io.SeekStart); err != nil {
			return 0, err
		}
		return replayV1Reader(f, apply)
	}
	if header != aofHeaderV2 && header != preRenameHeaderV2 {
		return 0, fmt.Errorf("invalid Amaquet AOF header")
	}
	pending := map[uint64]protocol.Request{}
	count := 0
	r := bufio.NewReaderSize(f, 64<<10)
	for {
		rec, ok, err := readRecord(r)
		if err != nil {
			return count, err
		}
		if !ok {
			return count, nil
		}
		switch rec.Kind {
		case "prepare":
			if rec.Request == nil {
				return count, fmt.Errorf("AOF prepare missing request")
			}
			pending[rec.TxID] = *rec.Request
		case "abort":
			delete(pending, rec.TxID)
		case "commit":
			req, ok := pending[rec.TxID]
			if !ok {
				return count, fmt.Errorf("AOF commit without prepare: %d", rec.TxID)
			}
			if err = apply(req); err != nil {
				return count, fmt.Errorf("replay AOF tx %d: %w", rec.TxID, err)
			}
			delete(pending, rec.TxID)
			count++
		default:
			return count, fmt.Errorf("unknown AOF record kind %q", rec.Kind)
		}
	}
}
func readRecord(r io.Reader) (Record, bool, error) {
	var rh [8]byte
	if _, err := io.ReadFull(r, rh[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return Record{}, false, nil
		}
		return Record{}, false, err
	}
	n := binary.BigEndian.Uint32(rh[0:4])
	if n > maxRecord {
		return Record{}, false, fmt.Errorf("AOF record exceeds limit")
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return Record{}, false, nil
		}
		return Record{}, false, err
	}
	if crc32.ChecksumIEEE(b) != binary.BigEndian.Uint32(rh[4:8]) {
		return Record{}, false, fmt.Errorf("AOF checksum mismatch")
	}
	var rec Record
	if err := json.Unmarshal(b, &rec); err != nil {
		return Record{}, false, err
	}
	return rec, true, nil
}
func replayV1Reader(r io.Reader, apply func(protocol.Request) error) (int, error) {
	br := bufio.NewReaderSize(r, 64<<10)
	count := 0
	for {
		var rh [8]byte
		if _, err := io.ReadFull(br, rh[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return count, nil
			}
			return count, err
		}
		n := binary.BigEndian.Uint32(rh[:4])
		if n > maxRecord {
			return count, fmt.Errorf("AOF record exceeds limit")
		}
		b := make([]byte, n)
		if _, err := io.ReadFull(br, b); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return count, nil
			}
			return count, err
		}
		if crc32.ChecksumIEEE(b) != binary.BigEndian.Uint32(rh[4:]) {
			return count, fmt.Errorf("AOF checksum mismatch")
		}
		var req protocol.Request
		if err := json.Unmarshal(b, &req); err != nil {
			return count, err
		}
		if err := apply(req); err != nil {
			return count, err
		}
		count++
	}
}
func migratePreRenameV2(path string) error {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	header := make([]byte, len(preRenameHeaderV2))
	if _, err = io.ReadFull(f, header); err != nil {
		_ = f.Close()
		return nil
	}
	if string(header) != preRenameHeaderV2 {
		_ = f.Close()
		return nil
	}

	tmp := path + ".header.tmp"
	_ = os.Remove(tmp)
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		_ = f.Close()
		return err
	}
	cleanup := func(e error) error {
		_ = f.Close()
		_ = out.Close()
		_ = os.Remove(tmp)
		return e
	}
	if _, err = out.WriteString(aofHeaderV2); err != nil {
		return cleanup(err)
	}
	if _, err = io.Copy(out, f); err != nil {
		return cleanup(err)
	}
	if err = f.Close(); err != nil {
		return cleanup(err)
	}
	if err = out.Sync(); err != nil {
		return cleanup(err)
	}
	if err = out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err = os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return syncDir(filepath.Dir(path))
}

func migrateV1(path string) error {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	h := make([]byte, len(aofHeaderV1))
	_, err = io.ReadFull(f, h)
	if err != nil {
		f.Close()
		return nil
	}
	if string(h) != aofHeaderV1 && string(h) != preRenameHeaderV1 {
		f.Close()
		return nil
	}
	reqs := []protocol.Request{}
	_, err = replayV1Reader(f, func(r protocol.Request) error { reqs = append(reqs, r); return nil })
	f.Close()
	if err != nil {
		return fmt.Errorf("migrate v1 AOF: %w", err)
	}
	tmp := path + ".v2.tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = out.WriteString(aofHeaderV2); err != nil {
		out.Close()
		return err
	}
	w := bufio.NewWriter(out)
	for i, req := range reqs {
		id := uint64(i + 1)
		for _, rec := range []Record{{Kind: "prepare", TxID: id, Request: &req, RecordedAtUnixNano: time.Now().UnixNano()}, {Kind: "commit", TxID: id}} {
			b, _ := json.Marshal(rec)
			var rh [8]byte
			binary.BigEndian.PutUint32(rh[:4], uint32(len(b)))
			binary.BigEndian.PutUint32(rh[4:], crc32.ChecksumIEEE(b))
			if _, err = w.Write(rh[:]); err != nil {
				out.Close()
				return err
			}
			if _, err = w.Write(b); err != nil {
				out.Close()
				return err
			}
		}
	}
	if err = w.Flush(); err != nil {
		out.Close()
		return err
	}
	if err = out.Sync(); err != nil {
		out.Close()
		return err
	}
	if err = out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
func scanV2(path string, fn func(Record) error) (uint64, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	h := make([]byte, len(aofHeaderV2))
	if _, err = io.ReadFull(f, h); err != nil {
		return 0, 0, err
	}
	if string(h) != aofHeaderV2 {
		return 0, 0, fmt.Errorf("not v2")
	}
	r := bufio.NewReader(f)
	var max uint64
	n := 0
	for {
		rec, ok, err := readRecord(r)
		if err != nil {
			return max, n, err
		}
		if !ok {
			return max, n, nil
		}
		if rec.TxID > max {
			max = rec.TxID
		}
		n++
		if fn != nil {
			if err = fn(rec); err != nil {
				return max, n, err
			}
		}
	}
}

func (a *AOF) Checkpoint(dst string) error {
	if strings.TrimSpace(dst) == "" {
		return fmt.Errorf("checkpoint destination required")
	}
	if err := a.Flush(); err != nil {
		return err
	}
	a.mu.Lock()
	src := a.file.Name()
	reqs := []protocol.Request{}
	_, err := Replay(src, func(r protocol.Request) error { reqs = append(reqs, r); return nil })
	a.mu.Unlock()
	if err != nil {
		return err
	}
	reqs = squashRequests(reqs)
	if err = os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	_ = os.Remove(tmp)
	if err = writeV2File(tmp, reqs); err != nil {
		return err
	}
	if _, err = Replay(tmp, func(protocol.Request) error { return nil }); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("checkpoint verification failed: %w", err)
	}
	if err = os.Rename(tmp, dst); err != nil {
		return err
	}
	return syncDir(filepath.Dir(dst))
}

func RestoreCheckpoint(src, dst string) error {
	if strings.TrimSpace(src) == "" || strings.TrimSpace(dst) == "" {
		return fmt.Errorf("source and destination are required")
	}
	if _, err := Replay(src, func(protocol.Request) error { return nil }); err != nil {
		return fmt.Errorf("checkpoint validation failed: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	tmp := dst + ".restore.tmp"
	_ = os.Remove(tmp)
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		in.Close()
		return err
	}
	_, err = io.Copy(out, in)
	_ = in.Close()
	if err == nil {
		err = out.Sync()
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err = migratePreRenameV2(tmp); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err = migrateV1(tmp); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if _, err = Replay(tmp, func(protocol.Request) error { return nil }); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	backup := dst + ".restore.bak"
	_ = os.Remove(backup)
	if _, e := os.Stat(dst); e == nil {
		if err = os.Rename(dst, backup); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	if err = os.Rename(tmp, dst); err != nil {
		_ = os.Rename(backup, dst)
		return err
	}
	if err = syncDir(filepath.Dir(dst)); err != nil {
		return err
	}
	_ = os.Remove(backup)
	return nil
}

func (a *AOF) Compact() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.flushLocked(); err != nil {
		return err
	}
	path := a.file.Name()
	reqs := []protocol.Request{}
	if _, err := Replay(path, func(r protocol.Request) error { reqs = append(reqs, r); return nil }); err != nil {
		return err
	}
	reqs = squashRequests(reqs)
	tmp := path + ".compact.tmp"
	backup := path + ".compact.bak"
	_ = os.Remove(tmp)
	_ = os.Remove(backup)
	if err := writeV2File(tmp, reqs); err != nil {
		return err
	}
	if _, err := Replay(tmp, func(protocol.Request) error { return nil }); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("compact verification failed: %w", err)
	}
	if err := a.file.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	rollback := func(cause error) error {
		_ = os.Remove(path)
		_ = os.Rename(backup, path)
		f, e := os.OpenFile(path, os.O_RDWR, 0600)
		if e == nil {
			_, _ = f.Seek(0, io.SeekEnd)
			a.file = f
			a.writer = bufio.NewWriterSize(f, 64<<10)
		}
		return cause
	}
	if err := os.Rename(path, backup); err != nil {
		f, e := os.OpenFile(path, os.O_RDWR, 0600)
		if e == nil {
			_, _ = f.Seek(0, io.SeekEnd)
			a.file = f
			a.writer = bufio.NewWriterSize(f, 64<<10)
		}
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return rollback(err)
	}
	if err := syncDir(filepath.Dir(path)); err != nil {
		return rollback(err)
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return rollback(err)
	}
	if _, err = f.Seek(0, io.SeekEnd); err != nil {
		_ = f.Close()
		return rollback(err)
	}
	a.file = f
	a.writer = bufio.NewWriterSize(f, 64<<10)
	a.next.Store(uint64(len(reqs)))
	a.healthMu.Lock()
	a.healthErr = nil
	a.healthMu.Unlock()
	_ = os.Remove(backup)
	_ = syncDir(filepath.Dir(path))
	return nil
}
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
func requestKey(r protocol.Request) (string, bool) {
	raw := r.Args["key"]
	if len(raw) == 0 {
		return "", false
	}
	var k string
	if json.Unmarshal(raw, &k) != nil || k == "" {
		return "", false
	}
	return k, true
}
func squashRequests(in []protocol.Request) []protocol.Request {
	for _, r := range in {
		if strings.EqualFold(strings.TrimSpace(r.Command), "BATCH") {
			return in
		}
	}
	out := make([]*protocol.Request, 0, len(in))
	byKey := map[string][]int{}
	type blobSeq struct {
		key  string
		reqs []protocol.Request
	}
	blobs := map[string]*blobSeq{}
	clearKey := func(k string) {
		for _, i := range byKey[k] {
			out[i] = nil
		}
		delete(byKey, k)
	}
	appendReq := func(r protocol.Request, k string) {
		cp := r
		out = append(out, &cp)
		if k != "" {
			byKey[k] = append(byKey[k], len(out)-1)
		}
	}
	uploadID := func(r protocol.Request) string {
		var id string
		_ = json.Unmarshal(r.Args["upload_id"], &id)
		return id
	}
	for _, r := range in {
		cmd := strings.ToUpper(strings.TrimSpace(r.Command))
		switch cmd {
		case "BLOB_BEGIN":
			id := uploadID(r)
			k, ok := requestKey(r)
			if id != "" && ok {
				blobs[id] = &blobSeq{key: k, reqs: []protocol.Request{r}}
			} else {
				appendReq(r, "")
			}
		case "BLOB_CHUNK":
			if b := blobs[uploadID(r)]; b != nil {
				b.reqs = append(b.reqs, r)
			} else {
				appendReq(r, "")
			}
		case "BLOB_ABORT":
			delete(blobs, uploadID(r))
		case "BLOB_COMMIT":
			id := uploadID(r)
			b := blobs[id]
			if b == nil {
				appendReq(r, "")
				continue
			}
			b.reqs = append(b.reqs, r)
			clearKey(b.key)
			for _, br := range b.reqs {
				appendReq(br, b.key)
			}
			delete(blobs, id)
		case "SET", "CREATE", "CAS":
			k, ok := requestKey(r)
			if !ok {
				appendReq(r, "")
				continue
			}
			clearKey(k)
			if cmd == "CAS" {
				r.Command = "SET"
				delete(r.Args, "expected_version")
			}
			appendReq(r, k)
		case "DEL":
			var keys []string
			_ = json.Unmarshal(r.Args["keys"], &keys)
			if len(keys) == 0 {
				appendReq(r, "")
				continue
			}
			for _, k := range keys {
				clearKey(k)
			}
		default:
			k, _ := requestKey(r)
			appendReq(r, k)
		}
	}

	res := make([]protocol.Request, 0, len(out))
	for _, r := range out {
		if r != nil {
			res = append(res, *r)
		}
	}
	return res
}

func writeV2File(path string, reqs []protocol.Request) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	w := bufio.NewWriterSize(f, 64<<10)
	if _, err = w.WriteString(aofHeaderV2); err != nil {
		f.Close()
		return err
	}
	writeRec := func(rec Record) error {
		b, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		var h [8]byte
		binary.BigEndian.PutUint32(h[:4], uint32(len(b)))
		binary.BigEndian.PutUint32(h[4:], crc32.ChecksumIEEE(b))
		if _, err = w.Write(h[:]); err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	}
	for i := range reqs {
		id := uint64(i + 1)
		req := reqs[i]
		if err = writeRec(Record{Kind: "prepare", TxID: id, Request: &req, RecordedAtUnixNano: time.Now().UnixNano()}); err != nil {
			f.Close()
			return err
		}
		if err = writeRec(Record{Kind: "commit", TxID: id}); err != nil {
			f.Close()
			return err
		}
	}
	if err = w.Flush(); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
