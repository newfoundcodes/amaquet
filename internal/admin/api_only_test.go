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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPServerDoesNotServeRoot(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	(&HTTPServer{}).Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("GET / status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestHTTPServerServesOpenAPI(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)

	(&HTTPServer{}).Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /openapi.json status = %d, want %d", recorder.Code, http.StatusOK)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("GET /openapi.json Content-Type = %q", contentType)
	}

	var document struct {
		OpenAPI string                                `json:"openapi"`
		Paths   map[string]map[string]json.RawMessage `json:"paths"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode OpenAPI document: %v", err)
	}

	if !strings.HasPrefix(document.OpenAPI, "3.1.") {
		t.Fatalf("OpenAPI version = %q, want 3.1.x", document.OpenAPI)
	}

	expected := map[string][]string{
		"/openapi.json":               {"get"},
		"/api/health":                 {"get"},
		"/api/ready":                  {"get"},
		"/metrics":                    {"get"},
		"/api/auth/check":             {"get"},
		"/api/session":                {"post", "delete"},
		"/api/identity/accept-invite": {"post"},
		"/api/identity/mfa":           {"post", "put", "delete"},
		"/api/overview":               {"get"},
		"/api/organization":           {"get", "put"},
		"/api/members":                {"get", "post"},
		"/api/members/{id}":           {"patch", "delete"},
		"/api/members/{id}/invite":    {"post"},
		"/api/rbac":                   {"get", "put"},
		"/api/api-keys":               {"get", "post"},
		"/api/api-keys/{id}":          {"delete"},
		"/api/system":                 {"get", "put"},
		"/api/persistence/checkpoint": {"post"},
		"/api/persistence/compact":    {"post"},
		"/api/data/keys":              {"get"},
		"/api/data/key":               {"get", "delete"},
		"/api/data/command":           {"post"},
		"/api/audit":                  {"get"},
	}

	for path, methods := range expected {
		item, ok := document.Paths[path]
		if !ok {
			t.Errorf("OpenAPI document missing path %s", path)
			continue
		}

		for _, method := range methods {
			if _, ok := item[method]; !ok {
				t.Errorf("OpenAPI document missing %s %s", strings.ToUpper(method), path)
			}
		}
	}
}

func TestOpenAPIEndpointMethods(t *testing.T) {
	server := (&HTTPServer{}).Handler()
	head := httptest.NewRecorder()

	server.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/openapi.json", nil))
	if head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("HEAD /openapi.json status/body = %d/%d", head.Code, head.Body.Len())
	}

	post := httptest.NewRecorder()
	server.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/openapi.json", nil))

	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /openapi.json status = %d, want %d", post.Code, http.StatusMethodNotAllowed)
	}

	if allow := post.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Fatalf("POST /openapi.json Allow = %q", allow)
	}
}
