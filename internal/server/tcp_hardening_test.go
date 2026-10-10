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
	"bufio"
	"context"
	"encoding/json"
	"io"
	"github.com/charmbracelet/log"
	"net"
	"testing"
	"time"

	"github.com/newfoundcodes/amaquet/internal/admin"
	"github.com/newfoundcodes/amaquet/internal/config"
	"github.com/newfoundcodes/amaquet/internal/core"
	"github.com/newfoundcodes/amaquet/internal/protocol"
	nt "github.com/newfoundcodes/amaquet/internal/types"
)

func TestCloseTerminatesIdleConnection(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	a, err := admin.NewState("", "")
	if err != nil {
		t.Fatal(err)
	}
	s := NewTCPServer(config.ProtocolConfig{Host: "127.0.0.1", Port: 0, ReadTimeoutMS: 60000}, NewDispatcher(e), a, nil, log.New(io.Discard), 10, 1<<20)
	client, server := net.Pipe()
	defer client.Close()
	s.connMu.Lock()
	s.conns[server] = struct{}{}
	s.connMu.Unlock()
	s.wg.Add(1)
	go func() { defer s.wg.Done(); s.serveConn(server) }()
	done := make(chan struct{})
	go func() { _ = s.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close blocked on idle connection")
	}
}

func TestPartialFrameHonorsReadDeadline(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	a, _ := admin.NewState("", "")
	s := NewTCPServer(config.ProtocolConfig{ReadTimeoutMS: 30}, NewDispatcher(e), a, nil, log.New(io.Discard), 10, 1<<20)
	client, server := net.Pipe()
	defer client.Close()
	done := make(chan struct{})
	go func() { s.serveConn(server); close(done) }()
	_, _ = client.Write([]byte("NF"))
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("partial frame held connection past read deadline")
	}
	_ = bufio.NewReader(client)
	_ = context.Background()
}

func TestInboundPayloadBudgetIsBounded(t *testing.T) {
	b := newByteBudget(64 << 20)
	r1, err := b.acquire(48 << 20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.acquire(32 << 20); err == nil {
		t.Fatal("expected payload budget rejection")
	}
	r1()
	r2, err := b.acquire(32 << 20)
	if err != nil {
		t.Fatal(err)
	}
	r2()
}

func writeTestCommand(t *testing.T, c net.Conn, id uint64, req protocol.Request) {
	t.Helper()
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if err = protocol.WriteFrame(c, protocol.Frame{Kind: protocol.KindRequest, Opcode: protocol.OpCommand, RequestID: id, Payload: b}); err != nil {
		t.Fatal(err)
	}
}

func readTestResponse(t *testing.T, c net.Conn) (protocol.Frame, protocol.Response) {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(time.Second))
	f, err := protocol.ReadFrame(c)
	if err != nil {
		t.Fatal(err)
	}
	var r protocol.Response
	if err = json.Unmarshal(f.Payload, &r); err != nil {
		t.Fatal(err)
	}
	return f, r
}

func blockingQueueDispatcher(t *testing.T) (*core.Engine, *Dispatcher) {
	t.Helper()
	e := core.NewEngine()
	d := NewDispatcher(e)
	if err := e.Set("q", core.Value{Type: core.TypeBlockingQueue, Data: nt.NewBlockingQueue()}, 0, false, false); err != nil {
		d.Close()
		e.Close()
		t.Fatal(err)
	}
	return e, d
}

func TestDuplicateActiveRequestIDRejected(t *testing.T) {
	e, d := blockingQueueDispatcher(t)
	defer e.Close()
	defer d.Close()
	a, _ := admin.NewState("", "")
	defer a.Close()
	s := NewTCPServer(config.ProtocolConfig{ReadTimeoutMS: 1000, WriteTimeoutMS: 1000}, d, a, nil, log.New(io.Discard), 10, 1<<20)
	client, server := net.Pipe()
	defer client.Close()
	go s.serveConn(server)

	blocking := protocol.Request{Command: "OP", Args: map[string]json.RawMessage{
		"key":       json.RawMessage(`"q"`),
		"operation": json.RawMessage(`"DEQUEUE"`),
	}}
	writeTestCommand(t, client, 77, blocking)
	writeTestCommand(t, client, 77, protocol.Request{Command: "PING"})
	f, r := readTestResponse(t, client)
	if f.RequestID != 77 || r.OK || r.Error == nil || r.Error.Code != "DUPLICATE_REQUEST" {
		t.Fatalf("expected duplicate request rejection, got frame=%+v response=%+v", f, r)
	}
}

func TestPerConnectionInflightLimitRejected(t *testing.T) {
	e, d := blockingQueueDispatcher(t)
	defer e.Close()
	defer d.Close()
	a, _ := admin.NewState("", "")
	defer a.Close()
	s := NewTCPServer(config.ProtocolConfig{ReadTimeoutMS: 1000, WriteTimeoutMS: 1000}, d, a, nil, log.New(io.Discard), 10, 1<<20)
	s.SetRuntimeLimits(8, 1, 2<<20, 20)
	client, server := net.Pipe()
	defer client.Close()
	go s.serveConn(server)

	blocking := protocol.Request{Command: "OP", Args: map[string]json.RawMessage{
		"key":       json.RawMessage(`"q"`),
		"operation": json.RawMessage(`"DEQUEUE"`),
	}}
	writeTestCommand(t, client, 1, blocking)
	writeTestCommand(t, client, 2, protocol.Request{Command: "PING"})
	f, r := readTestResponse(t, client)
	if f.RequestID != 2 || r.OK || r.Error == nil || r.Error.Code != "BUSY" {
		t.Fatalf("expected BUSY response, got frame=%+v response=%+v", f, r)
	}
}

func TestPubSubEventRefreshesIdleWriteDeadline(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	d := NewDispatcher(e)
	defer d.Close()
	a, _ := admin.NewState("", "")
	defer a.Close()
	ps := nt.NewPubSub()
	if err := e.Set("ps", core.Value{Type: core.TypePubSub, Data: ps}, 0, false, false); err != nil {
		t.Fatal(err)
	}
	s := NewTCPServer(config.ProtocolConfig{RequireAuth: false, ReadTimeoutMS: 1000, WriteTimeoutMS: 25}, d, a, nil, log.New(io.Discard), 10, 1<<20)
	client, server := net.Pipe()
	defer client.Close()
	go s.serveConn(server)
	b, _ := json.Marshal(map[string]any{"key": "ps", "channel": "c", "buffer": 8})
	if err := protocol.WriteFrame(client, protocol.Frame{Kind: protocol.KindRequest, Opcode: protocol.OpSubscribe, RequestID: 9, Payload: b}); err != nil {
		t.Fatal(err)
	}
	_, resp := readTestResponse(t, client)
	if !resp.OK {
		t.Fatalf("subscribe failed: %+v", resp)
	}
	time.Sleep(60 * time.Millisecond)
	ps.Publish("c", core.Value{Type: core.TypeUTF8String, Data: nt.UTF8String("hello")})
	_ = client.SetReadDeadline(time.Now().Add(time.Second))
	f, err := protocol.ReadFrame(client)
	if err != nil {
		t.Fatalf("idle event not delivered: %v", err)
	}
	if f.Kind != protocol.KindEvent {
		t.Fatalf("kind=%v", f.Kind)
	}
}

func TestExistingProtocolActorRejectedAfterRevocation(t *testing.T) {
	e := core.NewEngine()
	defer e.Close()
	d := NewDispatcher(e)
	defer d.Close()
	a, _ := admin.NewState("", "")
	defer a.Close()
	k, secret, err := a.CreateAPIKey("operator", "tcp", admin.RoleDeveloper, nil)
	if err != nil {
		t.Fatal(err)
	}
	s := NewTCPServer(config.ProtocolConfig{RequireAuth: true, ReadTimeoutMS: 1000, WriteTimeoutMS: 1000}, d, a, nil, log.New(io.Discard), 10, 1<<20)
	client, server := net.Pipe()
	defer client.Close()
	go s.serveConn(server)
	ab, _ := json.Marshal(map[string]any{"api_key": secret})
	if err = protocol.WriteFrame(client, protocol.Frame{Kind: protocol.KindRequest, Opcode: protocol.OpAuth, RequestID: 1, Payload: ab}); err != nil {
		t.Fatal(err)
	}
	_, r := readTestResponse(t, client)
	if !r.OK {
		t.Fatalf("auth failed: %+v", r)
	}
	if err = a.RevokeAPIKey("operator", k.ID); err != nil {
		t.Fatal(err)
	}
	writeTestCommand(t, client, 2, protocol.Request{Command: "PING"})
	_, r = readTestResponse(t, client)
	if r.OK || r.Error == nil || r.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("revoked actor still usable: %+v", r)
	}
}
