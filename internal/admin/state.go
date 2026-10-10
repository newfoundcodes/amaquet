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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Role string

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Member struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type MemberCredential struct {
	MemberID        string     `json:"member_id"`
	InviteHash      string     `json:"invite_hash,omitempty"`
	InviteExpiresAt *time.Time `json:"invite_expires_at,omitempty"`
	AccessHash      string     `json:"access_hash,omitempty"`
	TOTPSecret      string     `json:"totp_secret,omitempty"`
	MFAPending      bool       `json:"mfa_pending,omitempty"`
	MFAEnabled      bool       `json:"mfa_enabled,omitempty"`
}

type APIKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Hash       string     `json:"hash,omitempty"`
	Role       Role       `json:"role"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	Revoked    bool       `json:"revoked"`
}

type AuditEvent struct {
	ID       string         `json:"id"`
	At       time.Time      `json:"at"`
	Actor    string         `json:"actor"`
	Action   string         `json:"action"`
	Resource string         `json:"resource"`
	Details  map[string]any `json:"details,omitempty"`
}

type persisted struct {
	SchemaVersion     int                         `json:"schema_version"`
	BootstrapDisabled bool                        `json:"bootstrap_disabled,omitempty"`
	Organization      Organization                `json:"organization"`
	Members           map[string]Member           `json:"members"`
	MemberAuth        map[string]MemberCredential `json:"member_auth,omitempty"`
	APIKeys           map[string]APIKey           `json:"api_keys"`
	RBAC              map[Role][]string           `json:"rbac"`
	Audit             []AuditEvent                `json:"audit"`
}

type State struct {
	mu                  sync.RWMutex
	path                string
	bootstrapToken      string
	data                persisted
	lastUsedDirty       bool
	keyByHash           map[string]string
	memberByHash        map[string]string
	inviteByHash        map[string]string
	revocationListeners []func(string)
	stop                chan struct{}
	done                chan struct{}
	closeOnce           sync.Once
}

const (
	currentAdminSchemaVersion = 3
	preRenameDefaultOrgName   = "\x4e\x65\x77\x66\x6f\x75\x6e\x64\x44\x42"
	preRenameDefaultOrgSlug   = "\x6e\x65\x77\x66\x6f\x75\x6e\x64\x64\x62"
)

const (
	RoleAdmin     Role = "admin"
	RoleAuditor   Role = "auditor"
	RoleDeveloper Role = "developer"
)

var ValidRoles = []Role{RoleAdmin, RoleAuditor, RoleDeveloper}

func migratePersisted(p *persisted) error {
	if p.SchemaVersion > currentAdminSchemaVersion {
		return fmt.Errorf("admin state schema %d is newer than supported %d", p.SchemaVersion, currentAdminSchemaVersion)
	}

	if p.SchemaVersion == 0 {
		p.SchemaVersion = 1
	}

	for p.SchemaVersion < currentAdminSchemaVersion {
		switch p.SchemaVersion {
		case 1:
			if p.MemberAuth == nil {
				p.MemberAuth = map[string]MemberCredential{}
			}
			p.SchemaVersion = 2
		case 2:
			if p.Organization.Name == preRenameDefaultOrgName && p.Organization.Slug == preRenameDefaultOrgSlug {
				p.Organization.Name = "Amaquet"
				p.Organization.Slug = "amaquet"
				p.Organization.UpdatedAt = time.Now().UTC()
			}
			p.SchemaVersion = 3
		default:
			return fmt.Errorf("no admin state migration from schema %d", p.SchemaVersion)
		}
	}

	return nil
}

func NewState(path, bootstrapToken string) (*State, error) {
	s := &State{
		path:           path,
		bootstrapToken: bootstrapToken,
		keyByHash:      map[string]string{},
		memberByHash:   map[string]string{},
		inviteByHash:   map[string]string{},
		stop:           make(chan struct{}),
		done:           make(chan struct{}),
	}

	s.data.SchemaVersion = currentAdminSchemaVersion
	s.data.Members = map[string]Member{}
	s.data.MemberAuth = map[string]MemberCredential{}
	s.data.APIKeys = map[string]APIKey{}
	s.data.RBAC = defaultRBAC()

	now := time.Now().UTC()
	s.data.Organization = Organization{
		ID:        newID("org"),
		Name:      "Amaquet",
		Slug:      "amaquet",
		CreatedAt: now,
		UpdatedAt: now,
	}

	stateMigrated := false
	if path != "" {
		if b, e := os.ReadFile(path); e == nil {
			if e = json.Unmarshal(b, &s.data); e != nil {
				return nil, e
			}

			previousSchemaVersion := s.data.SchemaVersion
			if e = migratePersisted(&s.data); e != nil {
				return nil, e
			}

			stateMigrated = s.data.SchemaVersion != previousSchemaVersion
		} else if !os.IsNotExist(e) {
			return nil, e
		}
	}

	if s.data.Members == nil {
		s.data.Members = map[string]Member{}
	}

	if s.data.APIKeys == nil {
		s.data.APIKeys = map[string]APIKey{}
	}

	if s.data.MemberAuth == nil {
		s.data.MemberAuth = map[string]MemberCredential{}
	}

	if s.data.RBAC == nil {
		s.data.RBAC = defaultRBAC()
	}

	if s.data.SchemaVersion == 0 {
		s.data.SchemaVersion = currentAdminSchemaVersion
	}

	for id, k := range s.data.APIKeys {
		if k.Hash != "" {
			s.keyByHash[k.Hash] = id
		}
	}

	for id, c := range s.data.MemberAuth {
		if c.AccessHash != "" {
			s.memberByHash[c.AccessHash] = id
		}

		if c.InviteHash != "" && c.InviteExpiresAt != nil && time.Now().UTC().Before(*c.InviteExpiresAt) {
			s.inviteByHash[c.InviteHash] = id
		}
	}

	if path != "" {
		if stateMigrated {
			if err := s.saveLocked(); err != nil {
				return nil, err
			}
		}

		go s.flushLoop()
	} else {
		close(s.done)
	}

	return s, nil
}

func (s *State) flushLoop() {
	t := time.NewTicker(5 * time.Second)

	defer t.Stop()
	defer close(s.done)

	for {
		select {
		case <-t.C:
			s.mu.Lock()
			if s.lastUsedDirty {
				_ = s.saveLocked()
			}
			s.mu.Unlock()
		case <-s.stop:
			s.mu.Lock()
			if s.lastUsedDirty {
				_ = s.saveLocked()
			}
			s.mu.Unlock()
			return
		}
	}
}

func (s *State) Close() {
	s.closeOnce.Do(func() {
		close(s.stop)
		<-s.done
	})
}

func (s *State) OnActorRevoked(fn func(string)) {
	if fn == nil {
		return
	}

	s.mu.Lock()
	s.revocationListeners = append(s.revocationListeners, fn)
	s.mu.Unlock()
}

func (s *State) notifyActorRevoked(actor string) {
	s.mu.RLock()
	ls := append([]func(string){}, s.revocationListeners...)
	s.mu.RUnlock()

	for _, fn := range ls {
		fn(actor)
	}
}

func defaultRBAC() map[Role][]string {
	return map[Role][]string{
		RoleAdmin: {"*"},
		RoleAuditor: {
			"data.read",
			"org.read",
			"members.read",
			"rbac.read",
			"keys.read",
			"system.read",
			"audit.read",
		},
		RoleDeveloper: {
			"data.read",
			"data.write",
			"org.read",
			"members.read",
			"system.read",
		},
	}
}

func newID(prefix string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)

	return prefix + "_" + hex.EncodeToString(b)
}

func (s *State) saveLocked() error {
	if s.path == "" {
		return nil
	}

	if e := os.MkdirAll(filepath.Dir(s.path), 0700); e != nil {
		return e
	}

	b, e := json.MarshalIndent(s.data, "", "  ")
	if e != nil {
		return e
	}

	tmp := s.path + ".tmp"
	f, e := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)

	if e != nil {
		return e
	}

	if _, e = f.Write(b); e != nil {
		_ = f.Close()
		return e
	}

	if e = f.Sync(); e != nil {
		_ = f.Close()
		return e
	}

	if e = f.Close(); e != nil {
		return e
	}

	if e = os.Rename(tmp, s.path); e != nil {
		return e
	}

	d, e := os.Open(filepath.Dir(s.path))
	if e == nil {
		_ = d.Sync()
		_ = d.Close()
	}

	s.lastUsedDirty = false
	return nil
}

func (s *State) auditLocked(actor, action, resource string, details map[string]any) {
	s.data.Audit = append(s.data.Audit, AuditEvent{
		ID:       newID("evt"),
		At:       time.Now().UTC(),
		Actor:    actor,
		Action:   action,
		Resource: resource,
		Details:  details,
	})

	if len(s.data.Audit) > 5000 {
		s.data.Audit = append([]AuditEvent(nil), s.data.Audit[len(s.data.Audit)-5000:]...)
	}
}

func validRole(r Role) bool {
	for _, v := range ValidRoles {
		if r == v {
			return true
		}
	}

	return false
}

func hasPerm(perms []string, p string) bool {
	for _, x := range perms {
		if x == "*" || x == p {
			return true
		}
	}

	return false
}

func (s *State) HasPermission(role Role, p string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return hasPerm(s.data.RBAC[role], p)
}

func (s *State) Authenticate(token string) (Role, string, bool) {
	if token == "" {
		return "", "", false
	}

	sum := sha256.Sum256([]byte(token))
	hexsum := hex.EncodeToString(sum[:])

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.data.BootstrapDisabled && s.bootstrapToken != "" {
		bs := sha256.Sum256([]byte(s.bootstrapToken))
		if subtle.ConstantTimeCompare(sum[:], bs[:]) == 1 {
			return RoleAdmin, "bootstrap", true
		}
	}

	if id, ok := s.keyByHash[hexsum]; ok {
		k, ok := s.data.APIKeys[id]
		if !ok || k.Revoked {
			return "", "", false
		}

		now := time.Now().UTC()
		if k.ExpiresAt != nil && !now.Before(*k.ExpiresAt) {
			return "", "", false
		}

		k.LastUsedAt = &now
		s.data.APIKeys[id] = k
		s.lastUsedDirty = true

		return k.Role, k.ID, true
	}

	if id, ok := s.memberByHash[hexsum]; ok {
		m, mok := s.data.Members[id]
		c := s.data.MemberAuth[id]

		if mok && m.Status == "active" && c.AccessHash != "" && !c.MFAEnabled {
			return m.Role, "member:" + id, true
		}
	}

	return "", "", false
}

func (s *State) ActorAuthorization(actor string) (Role, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if actor == "bootstrap" {
		return RoleAdmin, !s.data.BootstrapDisabled && s.bootstrapToken != ""
	}

	if strings.HasPrefix(actor, "member:") {
		id := strings.TrimPrefix(actor, "member:")
		m, ok := s.data.Members[id]

		if !ok || m.Status != "active" {
			return "", false
		}

		c, ok := s.data.MemberAuth[id]
		if !ok || c.AccessHash == "" {
			return "", false
		}

		return m.Role, true
	}

	k, ok := s.data.APIKeys[actor]
	if !ok || k.Revoked {
		return "", false
	}

	if k.ExpiresAt != nil && !time.Now().UTC().Before(*k.ExpiresAt) {
		return "", false
	}

	return k.Role, true
}

func (s *State) ValidateAPIKey(token string) (Role, string, bool) {
	return s.Authenticate(token)
}

func (s *State) Organization() Organization {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.data.Organization
}

func (s *State) UpdateOrganization(actor, name, slug string) (Organization, error) {
	name = strings.TrimSpace(name)
	slug = strings.TrimSpace(slug)

	if name == "" || slug == "" {
		return Organization{}, fmt.Errorf("name and slug are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Organization.Name = name
	s.data.Organization.Slug = slug
	s.data.Organization.UpdatedAt = time.Now().UTC()

	s.auditLocked(actor, "organization.update", s.data.Organization.ID, nil)
	return s.data.Organization, s.saveLocked()
}

func (s *State) Members() []Member {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Member, 0, len(s.data.Members))
	for _, m := range s.data.Members {
		out = append(out, m)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Email < out[j].Email
	})
	return out
}

func (s *State) AddMember(actor, name, email string, role Role) (Member, error) {
	if !validRole(role) {
		return Member{}, fmt.Errorf("invalid role")
	}

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return Member{}, fmt.Errorf("email required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range s.data.Members {
		if m.Email == email {
			return Member{}, fmt.Errorf("member already exists")
		}
	}

	m := Member{
		ID:        newID("mem"),
		Name:      strings.TrimSpace(name),
		Email:     email,
		Role:      role,
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}

	s.data.Members[m.ID] = m
	s.auditLocked(actor, "member.create", m.ID, map[string]any{"role": role})

	return m, s.saveLocked()
}

func (s *State) UpdateMember(actor, id string, role Role, status string) (Member, error) {
	if !validRole(role) {
		return Member{}, fmt.Errorf("invalid role")
	}
	s.mu.Lock()

	m, ok := s.data.Members[id]
	if !ok {
		s.mu.Unlock()
		return Member{}, fmt.Errorf("member not found")
	}

	oldStatus := m.Status
	m.Role = role

	if status != "" {
		m.Status = status
	}

	s.data.Members[id] = m
	s.auditLocked(actor, "member.update", id, map[string]any{"role": role, "status": m.Status})

	err := s.saveLocked()
	s.mu.Unlock()

	if err == nil && oldStatus != "disabled" && m.Status == "disabled" {
		s.notifyActorRevoked("member:" + id)
	}

	return m, err
}

func (s *State) DeleteMember(actor, id string) error {
	s.mu.Lock()
	if _, ok := s.data.Members[id]; !ok {
		s.mu.Unlock()
		return fmt.Errorf("member not found")
	}
	delete(s.data.Members, id)

	if c, ok := s.data.MemberAuth[id]; ok {
		delete(s.memberByHash, c.AccessHash)
		delete(s.inviteByHash, c.InviteHash)
		delete(s.data.MemberAuth, id)
	}
	s.auditLocked(actor, "member.delete", id, nil)

	err := s.saveLocked()
	s.mu.Unlock()

	if err == nil {
		s.notifyActorRevoked("member:" + id)
	}

	return err
}

func randomSecret(prefix string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	return prefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}

func (s *State) CreateMemberInvite(actor, id string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	secret, err := randomSecret("amaquet_invite_")
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.data.Members[id]
	if !ok {
		return "", fmt.Errorf("member not found")
	}

	if m.Status == "disabled" {
		return "", fmt.Errorf("member is disabled")
	}

	c := s.data.MemberAuth[id]
	if c.InviteHash != "" {
		delete(s.inviteByHash, c.InviteHash)
	}

	exp := time.Now().UTC().Add(ttl)
	c.MemberID = id
	c.InviteHash = hashSecret(secret)
	c.InviteExpiresAt = &exp
	s.data.MemberAuth[id] = c
	s.inviteByHash[c.InviteHash] = id

	s.auditLocked(actor, "member.invite", id, map[string]any{"expires_at": exp})
	return secret, s.saveLocked()
}

func (s *State) AcceptMemberInvite(invite string) (Member, string, error) {
	h := hashSecret(invite)
	s.mu.Lock()

	id, ok := s.inviteByHash[h]
	if !ok {
		s.mu.Unlock()
		return Member{}, "", fmt.Errorf("invalid invite")
	}

	c := s.data.MemberAuth[id]
	if c.InviteExpiresAt == nil || !time.Now().UTC().Before(*c.InviteExpiresAt) {
		delete(s.inviteByHash, h)

		c.InviteHash = ""
		c.InviteExpiresAt = nil
		s.data.MemberAuth[id] = c

		_ = s.saveLocked()
		s.mu.Unlock()

		return Member{}, "", fmt.Errorf("invite expired")
	}

	m, ok := s.data.Members[id]
	if !ok || m.Status == "disabled" {
		s.mu.Unlock()
		return Member{}, "", fmt.Errorf("member unavailable")
	}

	secret, err := randomSecret("amaquet_user_")
	if err != nil {
		s.mu.Unlock()
		return Member{}, "", err
	}

	if c.AccessHash != "" {
		delete(s.memberByHash, c.AccessHash)
	}

	c.AccessHash = hashSecret(secret)
	c.InviteHash = ""
	c.InviteExpiresAt = nil

	s.data.MemberAuth[id] = c
	delete(s.inviteByHash, h)

	s.memberByHash[c.AccessHash] = id
	s.auditLocked("member:"+id, "member.invite.accept", id, nil)

	err = s.saveLocked()
	s.mu.Unlock()

	if err != nil {
		return Member{}, "", err
	}

	s.notifyActorRevoked("member:" + id)
	return m, secret, nil
}

func totpCode(secret string, at time.Time) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return ""
	}

	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(at.Unix()/30))

	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(b[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f

	n := ((uint32(sum[off]) & 0x7f) << 24) | (uint32(sum[off+1]) << 16) | (uint32(sum[off+2]) << 8) | uint32(sum[off+3])
	return fmt.Sprintf("%06d", n%1000000)
}

func verifyTOTP(secret, code string, now time.Time) bool {
	if len(code) != 6 {
		return false
	}

	if _, err := strconv.Atoi(code); err != nil {
		return false
	}

	for d := -1; d <= 1; d++ {
		if subtle.ConstantTimeCompare([]byte(totpCode(secret, now.Add(time.Duration(d)*30*time.Second))), []byte(code)) == 1 {
			return true
		}
	}

	return false
}

func (s *State) AuthenticateInteractive(token, mfa string) (Role, string, bool) {
	if role, actor, ok := s.Authenticate(token); ok {
		return role, actor, true
	}

	h := hashSecret(token)
	s.mu.RLock()

	id, ok := s.memberByHash[h]
	if !ok {
		s.mu.RUnlock()
		return "", "", false
	}

	m, mok := s.data.Members[id]
	c := s.data.MemberAuth[id]
	s.mu.RUnlock()

	if !mok || m.Status != "active" || !c.MFAEnabled || !verifyTOTP(c.TOTPSecret, mfa, time.Now().UTC()) {
		return "", "", false
	}

	return m.Role, "member:" + id, true
}

func (s *State) BeginMemberMFA(actor string) (string, error) {
	if !strings.HasPrefix(actor, "member:") {
		return "", fmt.Errorf("member session required")
	}

	id := strings.TrimPrefix(actor, "member:")
	raw := make([]byte, 20)

	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data.Members[id]; !ok {
		return "", fmt.Errorf("member not found")
	}

	c := s.data.MemberAuth[id]
	c.TOTPSecret = secret
	c.MFAPending = true
	c.MFAEnabled = false
	s.data.MemberAuth[id] = c

	s.auditLocked(actor, "member.mfa.begin", id, nil)
	return secret, s.saveLocked()
}

func (s *State) ConfirmMemberMFA(actor, code string) error {
	if !strings.HasPrefix(actor, "member:") {
		return fmt.Errorf("member session required")
	}

	id := strings.TrimPrefix(actor, "member:")
	s.mu.Lock()

	c, ok := s.data.MemberAuth[id]
	if !ok || !c.MFAPending || c.TOTPSecret == "" {
		s.mu.Unlock()
		return fmt.Errorf("MFA setup not started")
	}

	if !verifyTOTP(c.TOTPSecret, code, time.Now().UTC()) {
		s.mu.Unlock()
		return fmt.Errorf("invalid MFA code")
	}

	c.MFAPending = false
	c.MFAEnabled = true
	s.data.MemberAuth[id] = c

	s.auditLocked(actor, "member.mfa.enable", id, nil)
	err := s.saveLocked()
	s.mu.Unlock()

	if err == nil {
		s.notifyActorRevoked(actor)
	}

	return err
}

func (s *State) DisableMemberMFA(actor, id string) error {
	s.mu.Lock()

	c, ok := s.data.MemberAuth[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("member credential not found")
	}

	c.TOTPSecret = ""
	c.MFAPending = false
	c.MFAEnabled = false
	s.data.MemberAuth[id] = c
	s.auditLocked(actor, "member.mfa.disable", id, nil)

	err := s.saveLocked()
	s.mu.Unlock()

	if err == nil {
		s.notifyActorRevoked("member:" + id)
	}

	return err
}

func (s *State) RBAC() map[Role][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := map[Role][]string{}
	for k, v := range s.data.RBAC {
		out[k] = append([]string(nil), v...)
	}

	return out
}

func (s *State) SetRBAC(actor string, rbac map[Role][]string) error {
	for _, r := range ValidRoles {
		if _, ok := rbac[r]; !ok {
			return fmt.Errorf("missing role %s", r)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.RBAC = rbac
	s.auditLocked(actor, "rbac.update", "rbac", nil)

	return s.saveLocked()
}

func (s *State) APIKeys() []APIKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]APIKey, 0, len(s.data.APIKeys))
	for _, k := range s.data.APIKeys {
		k.Hash = ""
		out = append(out, k)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

func (s *State) CreateAPIKey(actor, name string, role Role, expiresAt *time.Time) (APIKey, string, error) {
	if !validRole(role) {
		return APIKey{}, "", fmt.Errorf("invalid role")
	}

	raw := make([]byte, 32)
	if _, e := rand.Read(raw); e != nil {
		return APIKey{}, "", e
	}

	secret := "amaquet_" + base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(secret))

	k := APIKey{
		ID:        newID("key"),
		Name:      strings.TrimSpace(name),
		Prefix:    secret[:12],
		Hash:      hex.EncodeToString(sum[:]),
		Role:      role,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: expiresAt,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.APIKeys[k.ID] = k
	s.keyByHash[k.Hash] = k.ID

	if actor == "bootstrap" && role == RoleAdmin {
		s.data.BootstrapDisabled = true
	}

	s.auditLocked(actor, "api_key.create", k.ID, map[string]any{"role": role})
	if e := s.saveLocked(); e != nil {
		return APIKey{}, "", e
	}

	pub := k
	pub.Hash = ""

	return pub, secret, nil
}

func (s *State) RevokeAPIKey(actor, id string) error {
	s.mu.Lock()

	k, ok := s.data.APIKeys[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("API key not found")
	}

	k.Revoked = true
	s.data.APIKeys[id] = k
	s.auditLocked(actor, "api_key.revoke", id, nil)

	err := s.saveLocked()
	s.mu.Unlock()

	if err == nil {
		s.notifyActorRevoked(id)
	}

	return err
}

func (s *State) Audit(limit int) []AuditEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.data.Audit) {
		limit = len(s.data.Audit)
	}

	out := make([]AuditEvent, limit)
	copy(out, s.data.Audit[len(s.data.Audit)-limit:])

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	return out
}
