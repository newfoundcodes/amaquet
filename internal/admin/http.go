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

package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/newfoundcodes/amaquet/internal/config"
	"github.com/newfoundcodes/amaquet/internal/core"
	"github.com/newfoundcodes/amaquet/internal/protocol"
)

type CommandExecutor interface {
	Execute(context.Context, protocol.Request) (any, error)
}

type HTTPServer struct {
	State                 *State
	Engine                *core.Engine
	Executor              CommandExecutor
	Config                *config.Manager
	PersistenceHealth     func() error
	PersistenceCheckpoint func(string) (string, error)
	PersistenceCompact    func() error
	sessionMu             sync.Mutex
	sessions              map[string]sessionAuth
	MetricsRequireAuth    bool
	MaxSessions           int
	AuthFailuresPerMin    int
	authMu                sync.Mutex
	authFailures          map[string]authAttempt
	authLastPrune         time.Time
}

type authAttempt struct {
	Start    time.Time
	Failures int
}

type sessionAuth struct {
	Role      Role
	Actor     string
	ExpiresAt time.Time
}

type authContext struct {
	Role  Role
	Actor string
}

type contextKey int

const authKey contextKey = 1
const requestKey contextKey = 2

func (s *HTTPServer) InvalidateActorSessions(actor string) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()

	for id, sess := range s.sessions {
		if sess.Actor == actor {
			delete(s.sessions, id)
		}
	}
}

func (s *HTTPServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/openapi.json", serveOpenAPI)
	mux.HandleFunc("/api/health", s.health)
	mux.HandleFunc("/api/ready", s.ready)

	if s.MetricsRequireAuth {
		mux.HandleFunc("/metrics", s.withAuth("system.read", s.metrics))
	} else {
		mux.HandleFunc("/metrics", s.metrics)
	}

	mux.HandleFunc("/api/auth/check", s.authCheck)
	mux.HandleFunc("/api/session", s.session)

	mux.HandleFunc("/api/identity/accept-invite", s.acceptInvite)
	mux.HandleFunc("/api/identity/mfa", s.withAuth("org.read", s.memberMFA))

	mux.HandleFunc("/api/overview", s.withAuth("data.read", s.overview))
	mux.HandleFunc("/api/organization", s.withAuthByMethod(map[string]string{"GET": "org.read", "PUT": "org.write"}, s.organization))

	mux.HandleFunc("/api/members", s.withAuthByMethod(map[string]string{"GET": "members.read", "POST": "members.write"}, s.members))
	mux.HandleFunc("/api/members/", s.withAuth("members.write", s.member))

	mux.HandleFunc("/api/rbac", s.withAuthByMethod(map[string]string{"GET": "rbac.read", "PUT": "rbac.write"}, s.rbac))

	mux.HandleFunc("/api/api-keys", s.withAuthByMethod(map[string]string{"GET": "keys.read", "POST": "keys.write"}, s.apiKeys))
	mux.HandleFunc("/api/api-keys/", s.withAuth("keys.write", s.apiKey))

	mux.HandleFunc("/api/system", s.withAuthByMethod(map[string]string{"GET": "system.read", "PUT": "system.write"}, s.system))

	mux.HandleFunc("/api/persistence/checkpoint", s.withAuth("system.write", s.persistenceCheckpoint))
	mux.HandleFunc("/api/persistence/compact", s.withAuth("system.write", s.persistenceCompact))

	mux.HandleFunc("/api/data/keys", s.withAuth("data.read", s.dataKeys))
	mux.HandleFunc("/api/data/key", s.withAuthByMethod(map[string]string{"GET": "data.read", "DELETE": "data.write"}, s.dataKey))
	mux.HandleFunc("/api/data/command", s.withDynamicDataAuth(s.dataCommand))

	mux.HandleFunc("/api/audit", s.withAuth("audit.read", s.audit))

	return s.maintenance(securityHeaders(mux))
}

func (s *HTTPServer) maintenance(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.sessionMu.Lock()
		if s.sessions != nil {
			s.pruneSessionsLocked(time.Now())
		}

		s.sessionMu.Unlock()
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'")

		next.ServeHTTP(w, r)
	})
}

func requestIP(r *http.Request) string {
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return h
	}

	return r.RemoteAddr
}

func (s *HTTPServer) pruneSessionsLocked(now time.Time) {
	for id, v := range s.sessions {
		if !now.Before(v.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
}

func (s *HTTPServer) pruneAuthFailuresLocked(now time.Time) {
	if len(s.authFailures) < 4096 {
		return
	}

	for ip, a := range s.authFailures {
		if a.Start.IsZero() || now.Sub(a.Start) >= time.Minute {
			delete(s.authFailures, ip)
		}
	}

	for len(s.authFailures) >= 8192 {
		for ip := range s.authFailures {
			delete(s.authFailures, ip)
			break
		}
	}
}

func (s *HTTPServer) allowAuthAttempt(r *http.Request) bool {
	limit := s.AuthFailuresPerMin
	if limit < 1 {
		limit = 20
	}

	ip := requestIP(r)
	now := time.Now()

	s.authMu.Lock()
	defer s.authMu.Unlock()

	if s.authFailures == nil {
		s.authFailures = map[string]authAttempt{}
	}
	s.pruneAuthFailuresLocked(now)

	a := s.authFailures[ip]
	if a.Start.IsZero() || now.Sub(a.Start) >= time.Minute {
		delete(s.authFailures, ip)
		return true
	}

	return a.Failures < limit
}

func (s *HTTPServer) authFailed(r *http.Request) {
	limit := s.AuthFailuresPerMin
	_ = limit

	ip := requestIP(r)
	now := time.Now()
	s.authMu.Lock()

	if s.authFailures == nil {
		s.authFailures = map[string]authAttempt{}
	}
	s.pruneAuthFailuresLocked(now)

	a := s.authFailures[ip]
	if a.Start.IsZero() || now.Sub(a.Start) >= time.Minute {
		a = authAttempt{Start: now}
	}

	a.Failures++
	s.authFailures[ip] = a

	s.authMu.Unlock()
}

func (s *HTTPServer) authSucceeded(r *http.Request) {
	s.authMu.Lock()
	delete(s.authFailures, requestIP(r))
	s.authMu.Unlock()
}

func (s *HTTPServer) auth(r *http.Request) (authContext, bool) {
	if c, err := r.Cookie("amaquet_session"); err == nil && c.Value != "" {
		s.sessionMu.Lock()
		s.pruneSessionsLocked(time.Now())

		sess, ok := s.sessions[c.Value]
		if ok && time.Now().Before(sess.ExpiresAt) {
			role, valid := s.State.ActorAuthorization(sess.Actor)
			if valid {
				sess.Role = role
				s.sessions[c.Value] = sess

				s.sessionMu.Unlock()
				return authContext{role, sess.Actor}, true
			}

			delete(s.sessions, c.Value)
			ok = false
		}

		if ok {
			delete(s.sessions, c.Value)
		}

		s.sessionMu.Unlock()
	}

	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if token == "" {
		token = r.Header.Get("X-Amaquet-Admin-Token")
	}

	if !s.allowAuthAttempt(r) {
		return authContext{}, false
	}

	role, actor, ok := s.State.Authenticate(token)
	if !ok {
		s.authFailed(r)
		return authContext{}, false
	}

	s.authSucceeded(r)
	return authContext{role, actor}, true
}

func (s *HTTPServer) session(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		if !s.allowAuthAttempt(r) {
			writeError(w, 429, "RATE_LIMITED", "too many failed authentication attempts")
			return
		}

		var in struct {
			Token   string `json:"token"`
			MFACode string `json:"mfa_code,omitempty"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}

		role, actor, ok := s.State.AuthenticateInteractive(in.Token, in.MFACode)
		if !ok {
			s.authFailed(r)
			writeError(w, 401, "UNAUTHORIZED", "invalid token")
			return
		}
		s.authSucceeded(r)

		var raw [32]byte
		if _, err := rand.Read(raw[:]); err != nil {
			writeError(w, 500, "RANDOM_FAILED", err.Error())
			return
		}

		id := base64.RawURLEncoding.EncodeToString(raw[:])
		exp := time.Now().Add(12 * time.Hour)
		s.sessionMu.Lock()

		if s.sessions == nil {
			s.sessions = map[string]sessionAuth{}
		}
		s.pruneSessionsLocked(time.Now())

		maxSessions := s.MaxSessions
		if maxSessions < 1 {
			maxSessions = 10000
		}

		if len(s.sessions) >= maxSessions {
			s.sessionMu.Unlock()
			writeError(w, 503, "SESSION_LIMIT", "session capacity reached")
			return
		}

		s.sessions[id] = sessionAuth{role, actor, exp}
		s.sessionMu.Unlock()

		http.SetCookie(w, &http.Cookie{
			Name:     "amaquet_session",
			Value:    id,
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteStrictMode,
			Expires:  exp,
			MaxAge:   int((12 * time.Hour).Seconds()),
		})
		writeJSON(w, 200, map[string]any{"ok": true, "role": role, "expires_at": exp})

	case "DELETE":
		if c, err := r.Cookie("amaquet_session"); err == nil {
			s.sessionMu.Lock()
			delete(s.sessions, c.Value)
			s.sessionMu.Unlock()
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "amaquet_session",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   -1,
		})
		w.WriteHeader(204)

	default:
		writeError(w, 405, "METHOD_NOT_ALLOWED", "POST or DELETE required")
	}
}

func (s *HTTPServer) acceptInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "METHOD_NOT_ALLOWED", "POST required")
		return
	}

	var in struct {
		Invite string `json:"invite"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}

	m, secret, err := s.State.AcceptMemberInvite(in.Invite)
	if err != nil {
		writeError(w, 401, "INVALID_INVITE", err.Error())
		return
	}

	writeJSON(w, 201, map[string]any{
		"member":       m,
		"access_token": secret,
		"warning":      "The access token is shown only once.",
	})
}

func (s *HTTPServer) memberMFA(w http.ResponseWriter, r *http.Request) {
	ac, _ := r.Context().Value(authKey).(authContext)
	if !strings.HasPrefix(ac.Actor, "member:") {
		writeError(w, 403, "FORBIDDEN", "member session required")
		return
	}

	switch r.Method {
	case http.MethodPost:
		secret, err := s.State.BeginMemberMFA(ac.Actor)
		if err != nil {
			writeError(w, 400, "MFA_ERROR", err.Error())
			return
		}

		writeJSON(w, 201, map[string]any{
			"secret":  secret,
			"otpauth": "otpauth://totp/Amaquet?secret=" + secret + "&issuer=Amaquet",
		})

	case http.MethodPut:
		var in struct {
			Code string `json:"code"`
		}

		if !decodeJSON(w, r, &in) {
			return
		}

		if err := s.State.ConfirmMemberMFA(ac.Actor, in.Code); err != nil {
			writeError(w, 400, "MFA_ERROR", err.Error())
			return
		}

		writeJSON(w, 200, map[string]any{"enabled": true})

	case http.MethodDelete:
		id := strings.TrimPrefix(ac.Actor, "member:")
		if err := s.State.DisableMemberMFA(ac.Actor, id); err != nil {
			writeError(w, 400, "MFA_ERROR", err.Error())
			return
		}

		w.WriteHeader(204)

	default:
		writeError(w, 405, "METHOD_NOT_ALLOWED", "POST, PUT, or DELETE required")
	}
}

func (s *HTTPServer) withAuth(permission string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ac, ok := s.auth(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "valid bearer token required")
			return
		}

		if !s.State.HasPermission(ac.Role, permission) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "permission denied")
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), authKey, ac)))
	}
}

func (s *HTTPServer) withAuthByMethod(perms map[string]string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := perms[r.Method]
		if p == "" {
			writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			return
		}

		s.withAuth(p, next)(w, r)
	}
}

func (s *HTTPServer) withDynamicDataAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeError(w, 405, "METHOD_NOT_ALLOWED", "POST required")
			return
		}

		var req protocol.Request
		if e := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); e != nil {
			writeError(w, 400, "BAD_REQUEST", e.Error())
			return
		}

		r.Body = http.NoBody
		perm := "data.write"
		cmd := strings.ToUpper(req.Command)

		switch cmd {
		case "PING", "GET", "EXISTS", "TYPE", "TTL", "KEYS", "SCAN", "INFO", "TYPES", "MEMORY":
			perm = "data.read"
		case "OP":
			var op string
			if raw := req.Args["operation"]; len(raw) > 0 {
				_ = json.Unmarshal(raw, &op)
			}

			if adminReadOperation(strings.ToUpper(op)) {
				perm = "data.read"
			}
		}

		ac, ok := s.auth(r)
		if !ok {
			writeError(w, 401, "UNAUTHORIZED", "valid bearer token required")
			return
		}

		if !s.State.HasPermission(ac.Role, perm) {
			writeError(w, 403, "FORBIDDEN", "permission denied")
			return
		}

		ctx := context.WithValue(r.Context(), authKey, ac)
		ctx = context.WithValue(ctx, requestKey, req)
		next(w, r.WithContext(ctx))
	}
}

func actor(r *http.Request) string {
	if x, ok := r.Context().Value(authKey).(authContext); ok {
		return x.Actor
	}
	return "unknown"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": map[string]string{"code": code, "message": msg}})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	d.DisallowUnknownFields()

	if e := d.Decode(v); e != nil {
		writeError(w, 400, "BAD_REQUEST", e.Error())
		return false
	}

	return true
}

func (s *HTTPServer) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"ok": true, "service": "amaquet-admin", "time": time.Now().UTC()})
}

func (s *HTTPServer) ready(w http.ResponseWriter, r *http.Request) {
	if s.PersistenceHealth != nil {
		if err := s.PersistenceHealth(); err != nil {
			writeJSON(w, 503, map[string]any{"ok": false, "error": err.Error()})
			return
		}
	}

	writeJSON(w, 200, map[string]any{
		"ok":     true,
		"keys":   s.Engine.Len(),
		"memory": s.Engine.MemoryStats(),
	})
}

func (s *HTTPServer) metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	m := s.Engine.MemoryStats()
	c := s.Engine.CompressionStats()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	fmt.Fprintf(w, "# TYPE amaquet_keys gauge\namaquet_keys %d\n", s.Engine.Len())
	fmt.Fprintf(w, "# TYPE amaquet_memory_bytes gauge\namaquet_memory_bytes %d\n", m.UsedBytes)
	fmt.Fprintf(w, "# TYPE amaquet_memory_limit_bytes gauge\namaquet_memory_limit_bytes %d\n", m.MaxBytes)
	fmt.Fprintf(w, "# TYPE amaquet_evicted_keys_total counter\namaquet_evicted_keys_total %d\n", m.Evicted)
	fmt.Fprintf(w, "# TYPE amaquet_compressed_keys gauge\namaquet_compressed_keys %d\n", c.CompressedKeys)
	fmt.Fprintf(w, "# TYPE amaquet_compression_saved_bytes gauge\namaquet_compression_saved_bytes %d\n", c.SavingsBytes)
	fmt.Fprintf(w, "# TYPE amaquet_go_heap_bytes gauge\namaquet_go_heap_bytes %d\n", ms.HeapAlloc)
	fmt.Fprintf(w, "# TYPE amaquet_go_goroutines gauge\namaquet_go_goroutines %d\n", runtime.NumGoroutine())

	ready := 1
	if s.PersistenceHealth != nil && s.PersistenceHealth() != nil {
		ready = 0
	}

	fmt.Fprintf(w, "# TYPE amaquet_ready gauge\namaquet_ready %d\n", ready)
}

func (s *HTTPServer) authCheck(w http.ResponseWriter, r *http.Request) {
	ac, ok := s.auth(r)
	if !ok {
		writeError(w, 401, "UNAUTHORIZED", "invalid token")
		return
	}

	writeJSON(w, 200, map[string]any{"ok": true, "role": ac.Role, "actor": ac.Actor})
}

func (s *HTTPServer) overview(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"keys":         s.Engine.Len(),
		"organization": s.State.Organization(),
		"members":      len(s.State.Members()),
		"api_keys":     len(s.State.APIKeys()),
		"types":        len(core.AllDataTypes()),
		"compression":  s.Engine.CompressionStats(),
		"memory":       s.Engine.MemoryStats(),
	})
}

func (s *HTTPServer) organization(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		writeJSON(w, 200, s.State.Organization())
		return
	}

	var in struct{ Name, Slug string }
	if !decodeJSON(w, r, &in) {
		return
	}

	v, e := s.State.UpdateOrganization(actor(r), in.Name, in.Slug)
	if e != nil {
		writeError(w, 400, "INVALID", e.Error())
		return
	}

	writeJSON(w, 200, v)
}

func (s *HTTPServer) members(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		writeJSON(w, 200, s.State.Members())
		return
	}

	var in struct {
		Name, Email string
		Role        Role
	}

	if !decodeJSON(w, r, &in) {
		return
	}

	v, e := s.State.AddMember(actor(r), in.Name, in.Email, in.Role)
	if e != nil {
		writeError(w, 400, "INVALID", e.Error())
		return
	}

	writeJSON(w, 201, v)
}

func (s *HTTPServer) member(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/members/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 0 || parts[0] == "" {
		writeError(w, 400, "BAD_REQUEST", "member ID required")
		return
	}

	id := parts[0]
	if len(parts) == 2 && parts[1] == "invite" {
		if r.Method != http.MethodPost {
			writeError(w, 405, "METHOD_NOT_ALLOWED", "POST required")
			return
		}

		var in struct {
			TTLHours int `json:"ttl_hours"`
		}

		_ = json.NewDecoder(r.Body).Decode(&in)
		ttl := 24 * time.Hour

		if in.TTLHours > 0 {
			ttl = time.Duration(in.TTLHours) * time.Hour
		}

		secret, e := s.State.CreateMemberInvite(actor(r), id, ttl)
		if e != nil {
			writeError(w, 400, "INVALID", e.Error())
			return
		}

		writeJSON(w, 201, map[string]any{
			"invite":             secret,
			"expires_in_seconds": int64(ttl.Seconds()),
			"warning":            "The invitation token is shown only once.",
		})
		return
	}

	if len(parts) != 1 {
		writeError(w, 404, "NOT_FOUND", "unknown member action")
		return
	}

	switch r.Method {
	case "PATCH":
		var in struct {
			Role   Role
			Status string
		}

		if !decodeJSON(w, r, &in) {
			return
		}

		v, e := s.State.UpdateMember(actor(r), id, in.Role, in.Status)
		if e != nil {
			writeError(w, 400, "INVALID", e.Error())
			return
		}

		writeJSON(w, 200, v)

	case "DELETE":
		if e := s.State.DeleteMember(actor(r), id); e != nil {
			writeError(w, 404, "NOT_FOUND", e.Error())
			return
		}

		w.WriteHeader(204)

	default:
		writeError(w, 405, "METHOD_NOT_ALLOWED", "PATCH or DELETE required")
	}
}

func (s *HTTPServer) rbac(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		writeJSON(w, 200, s.State.RBAC())
		return
	}

	var in map[Role][]string
	if !decodeJSON(w, r, &in) {
		return
	}

	if e := s.State.SetRBAC(actor(r), in); e != nil {
		writeError(w, 400, "INVALID", e.Error())
		return
	}

	writeJSON(w, 200, s.State.RBAC())
}

func (s *HTTPServer) apiKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		writeJSON(w, 200, s.State.APIKeys())
		return
	}

	var in struct {
		Name      string
		Role      Role
		ExpiresAt *time.Time `json:"expires_at"`
	}

	if !decodeJSON(w, r, &in) {
		return
	}

	k, secret, e := s.State.CreateAPIKey(actor(r), in.Name, in.Role, in.ExpiresAt)
	if e != nil {
		writeError(w, 400, "INVALID", e.Error())
		return
	}

	writeJSON(w, 201, map[string]any{
		"api_key": k,
		"secret":  secret,
		"warning": "The secret is shown only in this response.",
	})
}

func (s *HTTPServer) apiKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		writeError(w, 405, "METHOD_NOT_ALLOWED", "DELETE required")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/api-keys/")
	if e := s.State.RevokeAPIKey(actor(r), id); e != nil {
		writeError(w, 404, "NOT_FOUND", e.Error())
		return
	}

	w.WriteHeader(204)
}

func publicConfig(c config.Config) config.Config {
	c.Security.BootstrapToken = ""
	return c
}

func (s *HTTPServer) system(w http.ResponseWriter, r *http.Request) {
	cur := s.Config.Get()
	if r.Method == "GET" {
		writeJSON(w, 200, map[string]any{"config": publicConfig(cur), "restart_required": false})
		return
	}

	var next config.Config
	if !decodeJSON(w, r, &next) {
		return
	}

	next.Security.BootstrapToken = ""
	restart := !reflect.DeepEqual(publicConfig(next), publicConfig(cur))

	if e := s.Config.Update(next); e != nil {
		writeError(w, 500, "SAVE_FAILED", e.Error())
		return
	}

	writeJSON(w, 200, map[string]any{"config": publicConfig(next), "restart_required": restart})
}

func (s *HTTPServer) persistenceCheckpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, 405, "METHOD_NOT_ALLOWED", "POST required")
		return
	}

	if s.PersistenceCheckpoint == nil {
		writeError(w, 409, "DISABLED", "AOF persistence is not enabled")
		return
	}

	var in struct {
		Name string `json:"name"`
	}

	if r.Body != http.NoBody && !decodeJSON(w, r, &in) {
		return
	}

	path, err := s.PersistenceCheckpoint(in.Name)
	if err != nil {
		writeError(w, 500, "CHECKPOINT_FAILED", err.Error())
		return
	}

	writeJSON(w, 201, map[string]any{"path": path})
}

func (s *HTTPServer) persistenceCompact(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeError(w, 405, "METHOD_NOT_ALLOWED", "POST required")
		return
	}

	if s.PersistenceCompact == nil {
		writeError(w, 409, "DISABLED", "AOF persistence is not enabled")
		return
	}

	if err := s.PersistenceCompact(); err != nil {
		writeError(w, 500, "COMPACT_FAILED", err.Error())
		return
	}

	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *HTTPServer) dataKeys(w http.ResponseWriter, r *http.Request) {
	count := 250
	fmt.Sscanf(r.URL.Query().Get("count"), "%d", &count)

	keys, cursor := s.Engine.Scan(r.URL.Query().Get("cursor"), r.URL.Query().Get("prefix"), count)
	writeJSON(w, 200, map[string]any{"keys": keys, "cursor": cursor})
}

func (s *HTTPServer) dataKey(w http.ResponseWriter, r *http.Request) {
	k := r.URL.Query().Get("key")
	if k == "" {
		writeError(w, 400, "BAD_REQUEST", "key required")
		return
	}

	if r.Method == "DELETE" {
		_, e := s.Executor.Execute(r.Context(), protocol.Request{
			Command: "DEL",
			Args: map[string]json.RawMessage{
				"keys": json.RawMessage("[" + string(strconvQuote(k)) + "]"),
			},
		})

		if e != nil {
			writeError(w, 400, "COMMAND_ERROR", e.Error())
			return
		}

		w.WriteHeader(204)
		return
	}

	res, e := s.Executor.Execute(r.Context(), protocol.Request{
		Command: "GET",
		Args: map[string]json.RawMessage{
			"key": json.RawMessage(strconvQuote(k)),
		},
	})

	if e != nil {
		writeError(w, 404, "NOT_FOUND", e.Error())
		return
	}

	writeJSON(w, 200, res)
}

func strconvQuote(s string) []byte {
	b, _ := json.Marshal(s)
	return b
}

func adminReadOperation(op string) bool {
	switch op {
	case "GET",
		"LEN",
		"RANGE",
		"VALUES",
		"MEMBERS",
		"CONTAINS",
		"SCORE",
		"COUNT",
		"KEYS",
		"ENTRIES",
		"PEEK",
		"STATS",
		"PENDING",
		"READ",
		"GETBIT",
		"ESTIMATE",
		"ITEMS",
		"QUANTILE",
		"LAST",
		"AGGREGATE",
		"SNAPSHOT",
		"RADIUS",
		"BOX",
		"POLYGON",
		"SEARCH",
		"LOOKUP",
		"EXACT",
		"PREFIX",
		"NEIGHBORS",
		"LOAD",
		"HELD",
		"REMAINING",
		"DISTANCE",
		"DEQUANTIZE",
		"HOST",
		"SCHEME",
		"CARDINALITY",
		"CMP":
		return true
	}
	return false
}

func (s *HTTPServer) dataCommand(w http.ResponseWriter, r *http.Request) {
	req := r.Context().Value(requestKey).(protocol.Request)
	res, e := s.Executor.Execute(r.Context(), req)

	if e != nil {
		writeError(w, 400, "COMMAND_ERROR", e.Error())
		return
	}

	writeJSON(w, 200, map[string]any{"ok": true, "result": res})
}

func (s *HTTPServer) audit(w http.ResponseWriter, r *http.Request) {
	limit := 200
	fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit)
	writeJSON(w, 200, s.State.Audit(limit))
}
