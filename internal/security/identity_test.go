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

package security

import (
	"path/filepath"
	"testing"
)

func TestIdentityGenerateSaveLoadAndVerify(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("amaquet-nonce")
	sig := id.Sign(msg)
	if !Verify(id.PublicBase64(), msg, sig) {
		t.Fatal("signature did not verify")
	}
	if Verify(id.PublicBase64(), []byte("other"), sig) {
		t.Fatal("signature verified wrong message")
	}

	dir := t.TempDir()
	pub := filepath.Join(dir, "public.pem")
	priv := filepath.Join(dir, "private.pem")
	if err := Save(id, pub, priv); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(pub, priv)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Fingerprint() != id.Fingerprint() {
		t.Fatalf("fingerprint mismatch")
	}
	if !Verify(loaded.PublicBase64(), msg, loaded.Sign(msg)) {
		t.Fatal("loaded key did not verify")
	}
}
