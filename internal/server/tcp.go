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
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/newfoundcodes/amaquet/internal/admin"
	"github.com/newfoundcodes/amaquet/internal/config"
	"github.com/newfoundcodes/amaquet/internal/core"
	"github.com/newfoundcodes/amaquet/internal/protocol"
	"github.com/newfoundcodes/amaquet/internal/security"
	nt "github.com/newfoundcodes/amaquet/internal/types"
)

type byteBudget struct {
	mu   sync.Mutex
	max  int64
	used int64
}

func newByteBudget(max int64) *byteBudget {
	if max < 1 {
		max = protocol.MaxPayloadSize
	}
	return &byteBudget{max: max}
}
func (b *byteBudget) acquire(n int64) (func(), error) {
	if n <= 0 {
		return func() {}, nil
	}
	b.mu.Lock()
	if b.used+n > b.max {
		b.mu.Unlock()
		return nil, fmt.Errorf("server inbound payload budget exceeded")
	}
	b.used += n
	b.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			b.mu.Lock()
			b.used -= n
			if b.used < 0 {
				b.used = 0
			}
			b.mu.Unlock()
		})
	}, nil
}

type authWindow struct {
	start    time.Time
	failures int
}

type TCPServer struct {
	Config         config.ProtocolConfig
	Dispatcher     *Dispatcher
	Auth           *admin.State
	Identity       *security.Identity
	Logger         *log.Logger
	listener       net.Listener
	sem            chan struct{}
	maxPayload     int64
	wg             sync.WaitGroup
	closed         chan struct{}
	connMu         sync.Mutex
	conns          map[net.Conn]struct{}
	stateMu        sync.Mutex
	states         map[*connState]struct{}
	globalReqSem   chan struct{}
	inbound        *byteBudget
	perConnLimit   int
	authMu         sync.Mutex
	authFailures   map[string]authWindow
	authFailLimit  int
	authLastPrune  time.Time
	commandTimeout time.Duration
}
type connState struct {
	conn          net.Conn
	r             *bufio.Reader
	server        *TCPServer
	writeMu       sync.Mutex
	authMu        sync.Mutex
	authenticated bool
	role          admin.Role
	actor         string
	subs          map[string]nt.Subscription
	subMu         sync.Mutex
	reqMu         sync.Mutex
	cancels       map[uint64]context.CancelFunc
	reqWG         sync.WaitGroup
	reqSem        chan struct{}
	ownerID       string
}

func NewTCPServer(c config.ProtocolConfig, d *Dispatcher, a *admin.State, identity *security.Identity, logger *log.Logger, maxConnections int, maxPayload int64) *TCPServer {
	if logger == nil {
		logger = log.Default()
	}
	if maxConnections < 1 {
		maxConnections = 10000
	}
	if maxPayload <= 0 || maxPayload > protocol.MaxPayloadSize {
		maxPayload = protocol.MaxPayloadSize
	}
	return &TCPServer{Config: c, Dispatcher: d, Auth: a, Identity: identity, Logger: logger, sem: make(chan struct{}, maxConnections), maxPayload: maxPayload, closed: make(chan struct{}), conns: map[net.Conn]struct{}{}, states: map[*connState]struct{}{}, globalReqSem: make(chan struct{}, 4096), perConnLimit: 128, inbound: newByteBudget(256 << 20), authFailures: map[string]authWindow{}, authFailLimit: 20, commandTimeout: 30 * time.Second}
}

func (s *TCPServer) SetRuntimeLimits(globalRequests, perConnection int, inboundPayloadBytes int64, authFailuresPerMinute int) {
	if globalRequests < 1 {
		globalRequests = 4096
	}
	if perConnection < 1 {
		perConnection = 128
	}
	if inboundPayloadBytes < s.maxPayload {
		inboundPayloadBytes = s.maxPayload
	}
	if authFailuresPerMinute < 1 {
		authFailuresPerMinute = 20
	}
	s.globalReqSem = make(chan struct{}, globalRequests)
	s.perConnLimit = perConnection
	s.inbound = newByteBudget(inboundPayloadBytes)
	s.authFailLimit = authFailuresPerMinute
}
func (s *TCPServer) SetCommandTimeout(ms int) {
	if ms < 1 {
		ms = 30000
	}
	s.commandTimeout = time.Duration(ms) * time.Millisecond
}
func remoteHost(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	h, _, err := net.SplitHostPort(addr.String())
	if err == nil {
		return h
	}
	return addr.String()
}
func (s *TCPServer) pruneAuthFailuresLocked(now time.Time) {
	if len(s.authFailures) < 4096 {
		return
	}
	for k, w := range s.authFailures {
		if w.start.IsZero() || now.Sub(w.start) >= time.Minute {
			delete(s.authFailures, k)
		}
	}

	for len(s.authFailures) >= 8192 {
		for k := range s.authFailures {
			delete(s.authFailures, k)
			break
		}
	}
}

func (s *TCPServer) authAllowed(addr net.Addr) bool {
	h := remoteHost(addr)
	now := time.Now()
	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.pruneAuthFailuresLocked(now)
	if s.authLastPrune.IsZero() || now.Sub(s.authLastPrune) >= time.Minute || len(s.authFailures) > 4096 {
		for k, v := range s.authFailures {
			if now.Sub(v.start) >= time.Minute {
				delete(s.authFailures, k)
			}
		}
		s.authLastPrune = now
	}
	w := s.authFailures[h]
	if w.start.IsZero() || now.Sub(w.start) >= time.Minute {
		delete(s.authFailures, h)
		return true
	}
	return w.failures < s.authFailLimit
}
func (s *TCPServer) authFailed(addr net.Addr) {
	h := remoteHost(addr)
	now := time.Now()
	s.authMu.Lock()
	s.pruneAuthFailuresLocked(now)
	w := s.authFailures[h]
	if w.start.IsZero() || now.Sub(w.start) >= time.Minute {
		w = authWindow{start: now}
	}
	w.failures++
	s.authFailures[h] = w
	s.authMu.Unlock()
}
func (s *TCPServer) authSucceeded(addr net.Addr) {
	s.authMu.Lock()
	delete(s.authFailures, remoteHost(addr))
	s.authMu.Unlock()
}

func (s *TCPServer) InvalidateActor(actor string) {
	s.stateMu.Lock()
	states := make([]*connState, 0, len(s.states))
	for cs := range s.states {
		states = append(states, cs)
	}
	s.stateMu.Unlock()
	for _, cs := range states {
		_, a, ok := cs.authSnapshot()
		if ok && a == actor {
			_ = cs.conn.Close()
		}
	}
}

func (s *TCPServer) ListenAndServe() error {
	addr := net.JoinHostPort(s.Config.Host, fmt.Sprintf("%d", s.Config.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	if s.Config.TLS {
		cert, err := tls.LoadX509KeyPair(s.Config.TLSCertFile, s.Config.TLSKeyFile)
		if err != nil {
			ln.Close()
			return err
		}
		ln = tls.NewListener(ln, &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
	}
	s.listener = ln
	s.Logger.Info("Protocol server listening", "address", addr, "tls", s.Config.TLS)
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.closed:
				return nil
			default:
			}
			return err
		}
		select {
		case s.sem <- struct{}{}:
			s.connMu.Lock()
			s.conns[conn] = struct{}{}
			s.connMu.Unlock()
			s.wg.Add(1)
			go func() {
				defer func() { s.connMu.Lock(); delete(s.conns, conn); s.connMu.Unlock(); <-s.sem; s.wg.Done() }()
				s.serveConn(conn)
			}()
		default:
			_ = conn.Close()
		}
	}
}
func (s *TCPServer) Close() error {
	select {
	case <-s.closed:
		return nil
	default:
		close(s.closed)
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.connMu.Lock()
	for c := range s.conns {
		_ = c.Close()
	}
	s.connMu.Unlock()
	s.wg.Wait()
	return nil
}
func (s *TCPServer) serveConn(conn net.Conn) {
	defer conn.Close()
	owner := fmt.Sprintf("%p", conn)
	cs := &connState{conn: conn, r: bufio.NewReader(conn), server: s, authenticated: !s.Config.RequireAuth, role: admin.RoleAdmin, actor: "anonymous", subs: map[string]nt.Subscription{}, cancels: map[uint64]context.CancelFunc{}, reqSem: make(chan struct{}, s.perConnLimit), ownerID: owner}
	s.stateMu.Lock()
	s.states[cs] = struct{}{}
	s.stateMu.Unlock()
	defer func() {
		s.stateMu.Lock()
		delete(s.states, cs)
		s.stateMu.Unlock()
		cs.closeSubs()
		s.Dispatcher.AbortUploadsForOwner(owner)
	}()
	for {
		if ms := s.Config.ReadTimeoutMS; ms > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(time.Duration(ms) * time.Millisecond))
		}
		f, release, err := protocol.ReadFrameLimitBudget(cs.r, s.maxPayload, s.inbound.acquire)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.Logger.Debug("Connection read error", "remote", conn.RemoteAddr(), "error", err)
			}
			return
		}
		if release == nil {
			release = func() {}
		}
		if f.Kind != protocol.KindRequest {
			_ = cs.respond(f.RequestID, nil, fmt.Errorf("expected request frame"))
			release()
			continue
		}
		if f.Opcode == protocol.OpCancel {
			_ = cs.cancelRequest(f)
			release()
			continue
		}
		if f.Opcode == protocol.OpCommand {
			select {
			case cs.reqSem <- struct{}{}:
			default:
				_ = cs.respond(f.RequestID, nil, fmt.Errorf("too many in-flight requests on connection"))
				release()
				continue
			}
			select {
			case s.globalReqSem <- struct{}{}:
			default:
				<-cs.reqSem
				_ = cs.respond(f.RequestID, nil, fmt.Errorf("server is busy"))
				release()
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), s.commandTimeout)
			ctx = withUploadOwner(ctx, owner)
			cs.reqMu.Lock()
			if _, duplicate := cs.cancels[f.RequestID]; duplicate {
				cs.reqMu.Unlock()
				cancel()
				<-s.globalReqSem
				<-cs.reqSem
				_ = cs.respond(f.RequestID, nil, fmt.Errorf("duplicate active request id %d", f.RequestID))
				release()
				continue
			}
			cs.cancels[f.RequestID] = cancel
			cs.reqMu.Unlock()
			cs.reqWG.Add(1)
			go func(frame protocol.Frame, release func()) {
				defer cs.reqWG.Done()
				defer func() {
					cancel()
					cs.reqMu.Lock()
					delete(cs.cancels, frame.RequestID)
					cs.reqMu.Unlock()
					<-s.globalReqSem
					<-cs.reqSem
					release()
				}()
				_ = cs.handle(ctx, frame)
			}(f, release)
			continue
		}
		ctx, cancel := context.WithTimeout(withUploadOwner(context.Background(), owner), s.commandTimeout)
		err = cs.handle(ctx, f)
		cancel()
		release()
		if err != nil && errors.Is(err, net.ErrClosed) {
			return
		}
	}
}
func (c *connState) cancelRequest(f protocol.Frame) error {
	var p struct {
		RequestID uint64 `json:"request_id"`
	}
	if err := json.Unmarshal(f.Payload, &p); err != nil {
		return c.respond(f.RequestID, nil, err)
	}
	c.reqMu.Lock()
	cancel := c.cancels[p.RequestID]
	c.reqMu.Unlock()
	if cancel != nil {
		cancel()
		return c.respond(f.RequestID, map[string]any{"cancelled": true, "request_id": p.RequestID}, nil)
	}
	return c.respond(f.RequestID, map[string]any{"cancelled": false, "request_id": p.RequestID}, nil)
}

func (c *connState) revalidateAuth() bool {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	if !c.server.Config.RequireAuth && c.actor == "anonymous" {
		return true
	}
	if !c.authenticated {
		return false
	}
	role, ok := c.server.Auth.ActorAuthorization(c.actor)
	if !ok {
		c.authenticated = false
		return false
	}
	c.role = role
	return true
}
func (c *connState) authSnapshot() (admin.Role, string, bool) {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	return c.role, c.actor, c.authenticated
}

func (c *connState) handle(ctx context.Context, f protocol.Frame) error {
	switch f.Opcode {
	case protocol.OpHello:
		var hello struct {
			Nonce string `json:"nonce"`
		}
		_ = json.Unmarshal(f.Payload, &hello)
		var negotiation struct {
			MinProtocol int `json:"min_protocol"`
			MaxProtocol int `json:"max_protocol"`
		}
		_ = json.Unmarshal(f.Payload, &negotiation)
		if negotiation.MinProtocol == 0 {
			negotiation.MinProtocol = int(protocol.Version)
		}
		if negotiation.MaxProtocol == 0 {
			negotiation.MaxProtocol = int(protocol.Version)
		}
		if negotiation.MinProtocol > int(protocol.Version) || negotiation.MaxProtocol < int(protocol.Version) {
			return c.respond(f.RequestID, nil, fmt.Errorf("no compatible protocol version"))
		}
		out := map[string]any{"name": "Amaquet", "version": "0.1.0", "protocol": protocol.Version, "selected_protocol": protocol.Version, "capabilities": []string{"cancel", "batch", "cas", "scan", "chunked_blob", "pubsub"}, "auth_required": c.server.Config.RequireAuth}
		if c.server.Identity != nil {
			out["identity_public_key"] = c.server.Identity.PublicBase64()
			out["identity_fingerprint"] = c.server.Identity.Fingerprint()
			if hello.Nonce != "" {
				out["nonce_signature"] = c.server.Identity.Sign([]byte(hello.Nonce))
			}
		}
		return c.respond(f.RequestID, out, nil)
	case protocol.OpAuth:
		var p struct {
			APIKey string `json:"api_key"`
		}
		if e := json.Unmarshal(f.Payload, &p); e != nil {
			return c.respond(f.RequestID, nil, e)
		}
		if !c.server.authAllowed(c.conn.RemoteAddr()) {
			return c.respond(f.RequestID, nil, fmt.Errorf("authentication rate limit exceeded"))
		}
		role, actor, ok := c.server.Auth.ValidateAPIKey(p.APIKey)
		if !ok {
			c.server.authFailed(c.conn.RemoteAddr())
			return c.respond(f.RequestID, nil, fmt.Errorf("invalid API key"))
		}
		c.server.authSucceeded(c.conn.RemoteAddr())
		c.authMu.Lock()
		c.authenticated = true
		c.role = role
		c.actor = actor
		c.authMu.Unlock()
		return c.respond(f.RequestID, map[string]any{"authenticated": true, "role": role}, nil)
	case protocol.OpCommand:
		if !c.revalidateAuth() {
			return c.respond(f.RequestID, nil, fmt.Errorf("authentication required"))
		}
		req, e := protocol.DecodeRequest(f.Payload)
		if e != nil {
			return c.respond(f.RequestID, nil, e)
		}
		perm := commandPermission(req)
		role, _, _ := c.authSnapshot()
		if !c.server.Auth.HasPermission(role, perm) {
			return c.respond(f.RequestID, nil, fmt.Errorf("permission denied: %s", perm))
		}
		res, e := c.server.Dispatcher.Execute(ctx, req)
		return c.respond(f.RequestID, res, e)
	case protocol.OpSubscribe:
		return c.subscribe(f)
	case protocol.OpUnsubscribe:
		return c.unsubscribe(f)
	default:
		return c.respond(f.RequestID, nil, fmt.Errorf("unknown opcode %d", f.Opcode))
	}
}
func commandPermission(r protocol.Request) string {
	cmd := strings.ToUpper(r.Command)
	switch cmd {
	case "PING", "GET", "EXISTS", "TYPE", "TTL", "KEYS", "SCAN", "INFO", "TYPES", "MEMORY", "BLOB_READ":
		return "data.read"
	case "OP":
		var op string
		if raw := r.Args["operation"]; len(raw) > 0 {
			_ = json.Unmarshal(raw, &op)
		}
		if isReadOp(strings.ToUpper(op)) {
			return "data.read"
		}
		return "data.write"
	default:
		return "data.write"
	}
}
func (c *connState) respond(id uint64, result any, err error) error {
	resp := protocol.Response{OK: err == nil, Result: result}
	if err != nil {
		resp.Error = &protocol.Error{Code: errorCode(err), Message: err.Error()}
	}
	b, e := json.Marshal(resp)
	if e != nil {
		return e
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if ms := c.server.Config.WriteTimeoutMS; ms > 0 {
		_ = c.conn.SetWriteDeadline(time.Now().Add(time.Duration(ms) * time.Millisecond))
	}
	return protocol.WriteFrame(c.conn, protocol.Frame{Kind: protocol.KindResponse, Opcode: protocol.OpCommand, RequestID: id, Payload: b})
}
func errorCode(err error) string {
	switch {
	case errors.Is(err, core.ErrNotFound):
		return "NOT_FOUND"
	case errors.Is(err, core.ErrWrongType):
		return "WRONG_TYPE"
	case errors.Is(err, core.ErrExists):
		return "EXISTS"
	case errors.Is(err, core.ErrConflict):
		return "CONFLICT"
	case errors.Is(err, core.ErrMemoryLimit):
		return "MEMORY_LIMIT"
	case strings.Contains(err.Error(), "too many in-flight") || strings.Contains(err.Error(), "server is busy") || strings.Contains(err.Error(), "rate limit"):
		return "BUSY"
	case strings.Contains(err.Error(), "duplicate active request id"):
		return "DUPLICATE_REQUEST"
	case strings.Contains(err.Error(), "permission denied"):
		return "FORBIDDEN"
	case strings.Contains(err.Error(), "authentication"):
		return "UNAUTHORIZED"
	default:
		return "ERROR"
	}
}
func (c *connState) subscribe(f protocol.Frame) error {
	if !c.revalidateAuth() {
		return c.respond(f.RequestID, nil, fmt.Errorf("authentication required"))
	}
	role, _, _ := c.authSnapshot()
	if !c.server.Auth.HasPermission(role, "data.read") {
		return c.respond(f.RequestID, nil, fmt.Errorf("permission denied: data.read"))
	}
	var p struct {
		Key     string `json:"key"`
		Channel string `json:"channel"`
		Buffer  int    `json:"buffer"`
	}
	if e := json.Unmarshal(f.Payload, &p); e != nil {
		return c.respond(f.RequestID, nil, e)
	}
	v, _, e := c.server.Dispatcher.Engine.Get(p.Key)
	if e != nil {
		return c.respond(f.RequestID, nil, e)
	}
	ps, ok := v.Data.(*nt.PubSub)
	if !ok {
		return c.respond(f.RequestID, nil, core.ErrWrongType)
	}
	id := p.Key + "\x00" + p.Channel
	sub := ps.Subscribe(p.Channel, p.Buffer)
	c.subMu.Lock()
	if old, ok := c.subs[id]; ok {
		old.Close()
	}
	c.subs[id] = sub
	c.subMu.Unlock()
	go func(requestID uint64, s nt.Subscription) {
		for msg := range s.C {
			if !c.revalidateAuth() {
				return
			}
			b, _ := json.Marshal(msg)
			c.writeMu.Lock()
			if ms := c.server.Config.WriteTimeoutMS; ms > 0 {
				_ = c.conn.SetWriteDeadline(time.Now().Add(time.Duration(ms) * time.Millisecond))
			}
			e := protocol.WriteFrame(c.conn, protocol.Frame{Kind: protocol.KindEvent, Opcode: protocol.OpSubscribe, RequestID: requestID, Payload: b})
			c.writeMu.Unlock()
			if e != nil {
				return
			}
		}
	}(f.RequestID, sub)
	return c.respond(f.RequestID, map[string]any{"subscribed": true, "key": p.Key, "channel": p.Channel}, nil)
}
func (c *connState) unsubscribe(f protocol.Frame) error {
	var p struct {
		Key     string `json:"key"`
		Channel string `json:"channel"`
	}
	if e := json.Unmarshal(f.Payload, &p); e != nil {
		return c.respond(f.RequestID, nil, e)
	}
	id := p.Key + "\x00" + p.Channel
	c.subMu.Lock()
	sub, ok := c.subs[id]
	if ok {
		delete(c.subs, id)
		sub.Close()
	}
	c.subMu.Unlock()
	return c.respond(f.RequestID, map[string]any{"unsubscribed": ok}, nil)
}
func (c *connState) closeSubs() {
	c.reqMu.Lock()
	for id, cancel := range c.cancels {
		cancel()
		delete(c.cancels, id)
	}
	c.reqMu.Unlock()
	c.reqWG.Wait()
	c.subMu.Lock()
	defer c.subMu.Unlock()
	for id, s := range c.subs {
		s.Close()
		delete(c.subs, id)
	}
}
