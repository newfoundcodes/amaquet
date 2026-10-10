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
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
)

type Identity struct {
	Public  ed25519.PublicKey
	Private ed25519.PrivateKey
}

func Generate() (Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	return Identity{Public: pub, Private: priv}, err
}
func (i Identity) Fingerprint() string {
	h := sha256.Sum256(i.Public)
	return "ed25519:" + hex.EncodeToString(h[:])
}
func (i Identity) PublicBase64() string { return base64.RawStdEncoding.EncodeToString(i.Public) }
func (i Identity) Sign(message []byte) string {
	return base64.RawStdEncoding.EncodeToString(ed25519.Sign(i.Private, message))
}
func Verify(publicBase64 string, message []byte, signature string) bool {
	pub, e := base64.RawStdEncoding.DecodeString(publicBase64)
	if e != nil {
		return false
	}
	sig, e := base64.RawStdEncoding.DecodeString(signature)
	if e != nil {
		return false
	}
	return len(pub) == ed25519.PublicKeySize && ed25519.Verify(ed25519.PublicKey(pub), message, sig)
}
func Save(i Identity, publicPath, privatePath string) error {
	pubDER, e := x509.MarshalPKIXPublicKey(i.Public)
	if e != nil {
		return e
	}
	privDER, e := x509.MarshalPKCS8PrivateKey(i.Private)
	if e != nil {
		return e
	}
	if e = os.WriteFile(publicPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}), 0644); e != nil {
		return e
	}
	return os.WriteFile(privatePath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}), 0600)
}
func Load(publicPath, privatePath string) (Identity, error) {
	var out Identity
	pb, e := os.ReadFile(publicPath)
	if e != nil {
		return out, e
	}
	block, _ := pem.Decode(pb)
	if block == nil {
		return out, fmt.Errorf("invalid public PEM")
	}
	pk, e := x509.ParsePKIXPublicKey(block.Bytes)
	if e != nil {
		return out, e
	}
	pub, ok := pk.(ed25519.PublicKey)
	if !ok {
		return out, fmt.Errorf("public key is not Ed25519")
	}
	qb, e := os.ReadFile(privatePath)
	if e != nil {
		return out, e
	}
	block, _ = pem.Decode(qb)
	if block == nil {
		return out, fmt.Errorf("invalid private PEM")
	}
	sk, e := x509.ParsePKCS8PrivateKey(block.Bytes)
	if e != nil {
		return out, e
	}
	priv, ok := sk.(ed25519.PrivateKey)
	if !ok {
		return out, fmt.Errorf("private key is not Ed25519")
	}
	out.Public = pub
	out.Private = priv
	return out, nil
}
