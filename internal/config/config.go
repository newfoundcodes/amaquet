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

package config

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

type ProtocolConfig struct {
	Host              string `json:"host"`
	Port              int    `json:"port"`
	TLS               bool   `json:"tls"`
	TLSCertFile       string `json:"tls_cert_file,omitempty"`
	TLSKeyFile        string `json:"tls_key_file,omitempty"`
	RequireAuth       bool   `json:"require_auth"`
	AllowInsecureAuth bool   `json:"allow_insecure_auth,omitempty"`
	ReadTimeoutMS     int    `json:"read_timeout_ms,omitempty"`
	WriteTimeoutMS    int    `json:"write_timeout_ms,omitempty"`
}

type AdminConfig struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	TLS                bool   `json:"tls"`
	TLSCertFile        string `json:"tls_cert_file,omitempty"`
	TLSKeyFile         string `json:"tls_key_file,omitempty"`
	AllowInsecure      bool   `json:"allow_insecure,omitempty"`
	MetricsRequireAuth bool   `json:"metrics_require_auth"`
	MaxSessions        int    `json:"max_sessions,omitempty"`
	AuthFailuresPerMin int    `json:"auth_failures_per_minute,omitempty"`
}

type SecurityConfig struct {
	BootstrapToken string `json:"bootstrap_token,omitempty"`
	PublicKeyFile  string `json:"public_key_file,omitempty"`
	PrivateKeyFile string `json:"private_key_file,omitempty"`
}

type PersistenceConfig struct {
	AOFEnabled bool   `json:"aof_enabled"`
	AOFPath    string `json:"aof_path"`
	Fsync      string `json:"fsync"`
}

type LimitsConfig struct {
	MaxConnections           int    `json:"max_connections"`
	MaxPayloadBytes          int64  `json:"max_payload_bytes"`
	MaxMemoryBytes           int64  `json:"max_memory_bytes,omitempty"`
	EvictionPolicy           string `json:"eviction_policy,omitempty"`
	MaxInflightRequests      int    `json:"max_inflight_requests,omitempty"`
	MaxInflightPerConnection int    `json:"max_inflight_per_connection,omitempty"`
	MaxInflightPayloadBytes  int64  `json:"max_inflight_payload_bytes,omitempty"`
	MaxPendingUploadBytes    int64  `json:"max_pending_upload_bytes,omitempty"`
	UploadTTLMS              int    `json:"upload_ttl_ms,omitempty"`
	CommandTimeoutMS         int    `json:"command_timeout_ms,omitempty"`
}

type CompressionConfig struct {
	Enabled           bool    `json:"enabled"`
	Algorithm         string  `json:"algorithm"`
	MinBytes          int     `json:"min_bytes"`
	MinSavingsPercent float64 `json:"min_savings_percent"`
}

type Config struct {
	Protocol    ProtocolConfig    `json:"protocol"`
	Admin       AdminConfig       `json:"admin"`
	Security    SecurityConfig    `json:"security"`
	Persistence PersistenceConfig `json:"persistence"`
	Limits      LimitsConfig      `json:"limits"`
	Compression CompressionConfig `json:"compression"`
	DataDir     string            `json:"data_dir"`
}

func Default() Config {
	return Config{
		Protocol: ProtocolConfig{
			Host:           "127.0.0.1",
			Port:           13378,
			RequireAuth:    true,
			ReadTimeoutMS:  120000,
			WriteTimeoutMS: 30000,
		},
		Admin: AdminConfig{
			Host:               "127.0.0.1",
			Port:               13379,
			MetricsRequireAuth: true,
			MaxSessions:        10000,
			AuthFailuresPerMin: 20,
		},
		Persistence: PersistenceConfig{
			AOFEnabled: false,
			AOFPath:    "./data/amaquet.aof",
			Fsync:      "everysec",
		},
		Limits: LimitsConfig{
			MaxConnections:           10000,
			MaxPayloadBytes:          64 << 20,
			EvictionPolicy:           "noeviction",
			MaxInflightRequests:      4096,
			MaxInflightPerConnection: 128,
			MaxInflightPayloadBytes:  256 << 20,
			MaxPendingUploadBytes:    1 << 30,
			UploadTTLMS:              10 * 60 * 1000,
			CommandTimeoutMS:         30000,
		},
		Compression: CompressionConfig{
			Enabled:           true,
			Algorithm:         "auto",
			MinBytes:          1024,
			MinSavingsPercent: 8,
		},
		DataDir: "./data",
	}
}

func Load(path string) (Config, error) {
	c := Default()
	if path != "" {
		b, e := os.ReadFile(path)
		if e != nil && !os.IsNotExist(e) {
			return c, e
		}

		if e == nil {
			if e = json.Unmarshal(b, &c); e != nil {
				return c, e
			}
		}
	}

	applyEnv(&c)
	if err := Validate(c); err != nil {
		return c, err
	}

	return c, nil
}

func Validate(c Config) error {
	if c.Protocol.Port < 1 || c.Protocol.Port > 65535 || c.Admin.Port < 1 || c.Admin.Port > 65535 {
		return fmt.Errorf("invalid port")
	}

	if c.Limits.MaxConnections < 1 {
		return fmt.Errorf("max_connections must be positive")
	}

	if c.Limits.MaxPayloadBytes < 1 || c.Limits.MaxPayloadBytes > 64<<20 {
		return fmt.Errorf("max_payload_bytes must be between 1 and %d", 64<<20)
	}

	if c.Limits.MaxMemoryBytes < 0 {
		return fmt.Errorf("max_memory_bytes must be non-negative")
	}

	if c.Limits.MaxInflightRequests < 1 || c.Limits.MaxInflightPerConnection < 1 {
		return fmt.Errorf("inflight request limits must be positive")
	}

	if c.Limits.MaxInflightPayloadBytes < c.Limits.MaxPayloadBytes {
		return fmt.Errorf("max_inflight_payload_bytes must be at least max_payload_bytes")
	}

	if c.Limits.MaxPendingUploadBytes < 1 || c.Limits.UploadTTLMS < 1000 || c.Limits.CommandTimeoutMS < 1 {
		return fmt.Errorf("upload limits must be positive, upload_ttl_ms at least 1000, and command_timeout_ms positive")
	}

	if c.Admin.MaxSessions < 1 || c.Admin.AuthFailuresPerMin < 1 {
		return fmt.Errorf("admin session and authentication limits must be positive")
	}

	switch c.Limits.EvictionPolicy {
	case "", "noeviction", "allkeys-lru", "allkeys-lfu", "volatile-ttl":
	default:
		return fmt.Errorf("invalid eviction_policy")
	}

	if c.Persistence.Fsync != "always" && c.Persistence.Fsync != "everysec" && c.Persistence.Fsync != "no" {
		return fmt.Errorf("persistence.fsync must be always, everysec, or no")
	}

	if c.Protocol.TLS && (c.Protocol.TLSCertFile == "" || c.Protocol.TLSKeyFile == "") {
		return fmt.Errorf("TLS requires tls_cert_file and tls_key_file")
	}

	if c.Admin.TLS && (c.Admin.TLSCertFile == "" || c.Admin.TLSKeyFile == "") {
		return fmt.Errorf("admin TLS requires tls_cert_file and tls_key_file")
	}

	if c.Protocol.ReadTimeoutMS < 0 || c.Protocol.WriteTimeoutMS < 0 {
		return fmt.Errorf("protocol timeouts must be non-negative")
	}

	if c.Protocol.RequireAuth && !c.Protocol.TLS && !c.Protocol.AllowInsecureAuth && !isLoopbackHost(c.Protocol.Host) {
		return fmt.Errorf("authenticated Amaquet on a non-loopback address requires TLS or allow_insecure_auth=true")
	}

	if !c.Admin.TLS && !c.Admin.AllowInsecure && !isLoopbackHost(c.Admin.Host) {
		return fmt.Errorf("admin API on a non-loopback address requires TLS or allow_insecure=true")
	}

	switch c.Compression.Algorithm {
	case "auto", "lz4", "rle", "deflate-fast", "none":
	default:
		return fmt.Errorf("compression.algorithm must be auto, lz4, rle, deflate-fast, or none")
	}

	if c.Compression.MinBytes < 0 {
		return fmt.Errorf("compression.min_bytes must be non-negative")
	}

	if c.Compression.MinSavingsPercent < 0 || c.Compression.MinSavingsPercent >= 100 {
		return fmt.Errorf("compression.min_savings_percent must be >= 0 and < 100")
	}

	return nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func applyEnv(c *Config) {
	if v := os.Getenv("AMAQUET_HOST"); v != "" {
		c.Protocol.Host = v
	}

	if v := os.Getenv("AMAQUET_PORT"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			c.Protocol.Port = n
		}
	}

	if v := os.Getenv("AMAQUET_ADMIN_HOST"); v != "" {
		c.Admin.Host = v
	}

	if v := os.Getenv("AMAQUET_ADMIN_PORT"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			c.Admin.Port = n
		}
	}

	if v := os.Getenv("AMAQUET_ALLOW_INSECURE_AUTH"); v != "" {
		c.Protocol.AllowInsecureAuth = v == "1" || v == "true"
	}

	if v := os.Getenv("AMAQUET_ADMIN_ALLOW_INSECURE"); v != "" {
		c.Admin.AllowInsecure = v == "1" || v == "true"
	}

	if v := os.Getenv("AMAQUET_BOOTSTRAP_TOKEN"); v != "" {
		c.Security.BootstrapToken = v
	}

	if v := os.Getenv("AMAQUET_TLS_CERT"); v != "" {
		c.Protocol.TLSCertFile = v
		c.Protocol.TLS = true
	}

	if v := os.Getenv("AMAQUET_TLS_KEY"); v != "" {
		c.Protocol.TLSKeyFile = v
		c.Protocol.TLS = true
	}

	if v := os.Getenv("AMAQUET_COMPRESSION"); v != "" {
		c.Compression.Enabled = v != "0" && v != "false"
	}

	if v := os.Getenv("AMAQUET_COMPRESSION_ALGORITHM"); v != "" {
		c.Compression.Algorithm = v
	}

	if v := os.Getenv("AMAQUET_COMPRESSION_MIN_BYTES"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			c.Compression.MinBytes = n
		}
	}

	if v := os.Getenv("AMAQUET_AOF_ENABLED"); v != "" {
		c.Persistence.AOFEnabled = v == "1" || v == "true"
	}

	if v := os.Getenv("AMAQUET_AOF_PATH"); v != "" {
		c.Persistence.AOFPath = v
	}

	if v := os.Getenv("AMAQUET_AOF_FSYNC"); v != "" {
		c.Persistence.Fsync = v
	}

	if v := os.Getenv("AMAQUET_MAX_MEMORY_BYTES"); v != "" {
		if n, e := strconv.ParseInt(v, 10, 64); e == nil {
			c.Limits.MaxMemoryBytes = n
		}
	}

	if v := os.Getenv("AMAQUET_EVICTION_POLICY"); v != "" {
		c.Limits.EvictionPolicy = v
	}

	if v := os.Getenv("AMAQUET_MAX_CONNECTIONS"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			c.Limits.MaxConnections = n
		}
	}

	if v := os.Getenv("AMAQUET_MAX_INFLIGHT_REQUESTS"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			c.Limits.MaxInflightRequests = n
		}
	}

	if v := os.Getenv("AMAQUET_MAX_INFLIGHT_PER_CONNECTION"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			c.Limits.MaxInflightPerConnection = n
		}
	}

	if v := os.Getenv("AMAQUET_MAX_INFLIGHT_PAYLOAD_BYTES"); v != "" {
		if n, e := strconv.ParseInt(v, 10, 64); e == nil {
			c.Limits.MaxInflightPayloadBytes = n
		}
	}

	if v := os.Getenv("AMAQUET_MAX_PENDING_UPLOAD_BYTES"); v != "" {
		if n, e := strconv.ParseInt(v, 10, 64); e == nil {
			c.Limits.MaxPendingUploadBytes = n
		}
	}

	if v := os.Getenv("AMAQUET_UPLOAD_TTL_MS"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			c.Limits.UploadTTLMS = n
		}
	}

	if v := os.Getenv("AMAQUET_COMMAND_TIMEOUT_MS"); v != "" {
		if n, e := strconv.Atoi(v); e == nil {
			c.Limits.CommandTimeoutMS = n
		}
	}
}

func Save(path string, c Config) error {
	if err := Validate(c); err != nil {
		return err
	}

	if path == "" {
		return fmt.Errorf("config path required")
	}

	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}

	b, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		return e
	}

	tmp := path + ".tmp"
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

	if e = os.Rename(tmp, path); e != nil {
		return e
	}

	d, e := os.Open(filepath.Dir(path))
	if e == nil {
		_ = d.Sync()
		_ = d.Close()
	}

	return nil
}
