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
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateMigratesPreRenameDefaultOrganization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin.json")
	before := persisted{
		SchemaVersion: 2,
		Organization:  Organization{ID: "org_legacy", Name: preRenameDefaultOrgName, Slug: preRenameDefaultOrgSlug},
		Members:       map[string]Member{},
		MemberAuth:    map[string]MemberCredential{},
		APIKeys:       map[string]APIKey{},
		RBAC:          defaultRBAC(),
	}

	b, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}

	if err = os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}

	s, err := NewState(path, "bootstrap-test")
	if err != nil {
		t.Fatal(err)
	}

	if got := s.Organization(); got.Name != "Amaquet" || got.Slug != "amaquet" {
		t.Fatalf("organization was not migrated: %#v", got)
	}
	s.Close()

	b, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var after persisted
	if err = json.Unmarshal(b, &after); err != nil {
		t.Fatal(err)
	}

	if after.SchemaVersion != currentAdminSchemaVersion || after.Organization.Name != "Amaquet" {
		t.Fatalf("migration was not persisted: %#v", after.Organization)
	}
}

func TestStateOrganizationMembersRBACAndAPIKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin.json")
	s, err := NewState(path, "bootstrap-test")

	if err != nil {
		t.Fatal(err)
	}

	role, actor, ok := s.Authenticate("bootstrap-test")
	if !ok || role != RoleAdmin || actor != "bootstrap" {
		t.Fatalf("bootstrap auth failed")
	}

	org, err := s.UpdateOrganization("bootstrap", "Example Org", "example")
	if err != nil || org.Slug != "example" {
		t.Fatalf("organization update: %v", err)
	}

	m, err := s.AddMember("bootstrap", "Developer", "dev@example.com", RoleDeveloper)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = s.UpdateMember("bootstrap", m.ID, RoleAuditor, "active"); err != nil {
		t.Fatal(err)
	}

	exp := time.Now().Add(time.Hour)
	pub, secret, err := s.CreateAPIKey("bootstrap", "integration", RoleDeveloper, &exp)

	if err != nil {
		t.Fatal(err)
	}

	if pub.Hash != "" || secret == "" {
		t.Fatal("API key secret/hash exposure invariant failed")
	}

	role, actor, ok = s.ValidateAPIKey(secret)
	if !ok || role != RoleDeveloper || actor != pub.ID {
		t.Fatalf("API-key auth failed")
	}

	if !s.HasPermission(RoleDeveloper, "data.write") || s.HasPermission(RoleAuditor, "data.write") {
		t.Fatal("default RBAC failed")
	}

	if err = s.RevokeAPIKey("bootstrap", pub.ID); err != nil {
		t.Fatal(err)
	}

	if _, _, ok = s.ValidateAPIKey(secret); ok {
		t.Fatal("revoked key still valid")
	}

	s2, err := NewState(path, "bootstrap-test")
	if err != nil {
		t.Fatal(err)
	}

	if s2.Organization().Slug != "example" || len(s2.Members()) != 1 {
		t.Fatal("state did not persist")
	}

	if len(s2.Audit(100)) == 0 {
		t.Fatal("audit trail empty")
	}
}

func TestEmptyAuditReturnsCollection(t *testing.T) {
	s, err := NewState(filepath.Join(t.TempDir(), "admin.json"), "bootstrap-test")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	events := s.Audit(100)
	if events == nil {
		t.Fatal("empty audit must be a non-nil collection")
	}

	if len(events) != 0 {
		t.Fatalf("expected no audit events, got %d", len(events))
	}
}
