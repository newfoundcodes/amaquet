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
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/newfoundcodes/amaquet/internal/core"
)

func newHTTPTestServer(t *testing.T, maxSessions, failures int) (*HTTPServer, *State, *core.Engine) {
	t.Helper()
	st, err := NewState("", "bootstrap-http-test")

	if err != nil {
		t.Fatal(err)
	}

	e := core.NewEngine()
	s := &HTTPServer{
		State:              st,
		Engine:             e,
		MetricsRequireAuth: true,
		MaxSessions:        maxSessions,
		AuthFailuresPerMin: failures,
	}

	return s, st, e
}

func TestMetricsRequireAuthentication(t *testing.T) {
	s, st, e := newHTTPTestServer(t, 10, 10)

	defer st.Close()
	defer e.Close()

	h := s.Handler()
	r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated metrics to be 401, got %d", w.Code)
	}

	r = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	r.Header.Set("Authorization", "Bearer bootstrap-http-test")

	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "amaquet_keys") {
		t.Fatalf("expected authenticated metrics, code=%d body=%q", w.Code, w.Body.String())
	}
}

func TestEmptyAuditEndpointReturnsJSONArray(t *testing.T) {
	s, st, e := newHTTPTestServer(t, 10, 10)
	defer st.Close()
	defer e.Close()

	r := httptest.NewRequest(http.MethodGet, "/api/audit?limit=500", nil)
	r.Header.Set("Authorization", "Bearer bootstrap-http-test")

	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected audit endpoint to return 200, got %d", w.Code)
	}

	if strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("expected an empty JSON array, got %q", w.Body.String())
	}
}

func TestSessionCapacityIsBounded(t *testing.T) {
	s, st, e := newHTTPTestServer(t, 1, 10)

	defer st.Close()
	defer e.Close()

	h := s.Handler()
	body := []byte(`{"token":"bootstrap-http-test"}`)

	r := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewReader(body))
	r.RemoteAddr = "192.0.2.10:1234"

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("first session failed: %d %s", w.Code, w.Body.String())
	}

	r = httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewReader(body))
	r.RemoteAddr = "192.0.2.10:1235"

	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "SESSION_LIMIT") {
		t.Fatalf("expected bounded session rejection, got %d %s", w.Code, w.Body.String())
	}
}

func TestSessionAuthenticationRateLimit(t *testing.T) {
	s, st, e := newHTTPTestServer(t, 10, 2)

	defer st.Close()
	defer e.Close()

	h := s.Handler()
	bad := []byte(`{"token":"not-valid"}`)

	for i := 0; i < 2; i++ {
		r := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewReader(bad))
		r.RemoteAddr = "192.0.2.22:1234"

		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("failure %d expected 401, got %d", i, w.Code)
		}
	}

	r := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewReader(bad))
	r.RemoteAddr = "192.0.2.22:9999"

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected auth throttle, got %d %s", w.Code, w.Body.String())
	}
}

func TestRevokedAPIKeyInvalidatesExistingSession(t *testing.T) {
	s, st, e := newHTTPTestServer(t, 10, 10)

	defer st.Close()
	defer e.Close()

	h := s.Handler()
	key, secret, err := st.CreateAPIKey("operator", "browser", RoleAdmin, nil)

	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"token":"` + secret + `"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/session", bytes.NewReader(body))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("session: %d %s", w.Code, w.Body.String())
	}

	var cookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "amaquet_session" {
			cookie = c
		}
	}

	if cookie == nil {
		t.Fatal("session cookie missing")
	}

	r = httptest.NewRequest(http.MethodGet, "/api/auth/check", nil)
	r.AddCookie(cookie)

	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("pre-revoke auth=%d", w.Code)
	}

	if err = st.RevokeAPIKey("operator", key.ID); err != nil {
		t.Fatal(err)
	}

	r = httptest.NewRequest(http.MethodGet, "/api/auth/check", nil)
	r.AddCookie(cookie)

	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session remained valid: %d %s", w.Code, w.Body.String())
	}
}

func TestMemberInviteIdentityAndMFA(t *testing.T) {
	_, st, e := newHTTPTestServer(t, 10, 10)

	defer st.Close()
	defer e.Close()

	m, err := st.AddMember("admin", "Alice", "alice@example.com", RoleAuditor)
	if err != nil {
		t.Fatal(err)
	}

	invite, err := st.CreateMemberInvite("admin", m.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	m2, token, err := st.AcceptMemberInvite(invite)
	if err != nil || m2.ID != m.ID || token == "" {
		t.Fatalf("accept: member=%+v err=%v", m2, err)
	}

	role, actor, ok := st.Authenticate(token)
	if !ok || role != RoleAuditor || actor != "member:"+m.ID {
		t.Fatalf("member auth role=%s actor=%s ok=%v", role, actor, ok)
	}

	secret, err := st.BeginMemberMFA(actor)
	if err != nil {
		t.Fatal(err)
	}

	code := totpCode(secret, time.Now().UTC())
	if err = st.ConfirmMemberMFA(actor, code); err != nil {
		t.Fatal(err)
	}

	if _, _, ok = st.Authenticate(token); ok {
		t.Fatal("member token bypassed enabled MFA")
	}

	role, actor, ok = st.AuthenticateInteractive(token, totpCode(secret, time.Now().UTC()))
	if !ok || role != RoleAuditor {
		t.Fatalf("MFA auth failed role=%s actor=%s", role, actor)
	}
}

func TestMemberCredentialIndexesSurviveRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "admin.json")
	s, err := NewState(path, "bootstrap")

	if err != nil {
		t.Fatal(err)
	}

	m, err := s.AddMember("bootstrap", "Alice", "alice@example.test", RoleDeveloper)
	if err != nil {
		t.Fatal(err)
	}

	invite, err := s.CreateMemberInvite("bootstrap", m.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	_, token, err := s.AcceptMemberInvite(invite)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()

	s2, err := NewState(path, "bootstrap")
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()

	role, actor, ok := s2.Authenticate(token)
	if !ok || role != RoleDeveloper || actor != "member:"+m.ID {
		t.Fatalf("member auth after restart = %v %q %q", ok, role, actor)
	}

	if got, ok := s2.ActorAuthorization(actor); !ok || got != RoleDeveloper {
		t.Fatalf("member actor authorization after restart = %v %q", ok, got)
	}
}
