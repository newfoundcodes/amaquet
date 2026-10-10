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

package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/newfoundcodes/amaquet/internal/admin"
	"github.com/newfoundcodes/amaquet/internal/compression"
	"github.com/newfoundcodes/amaquet/internal/config"
	"github.com/newfoundcodes/amaquet/internal/core"
	"github.com/newfoundcodes/amaquet/internal/persistence"
	"github.com/newfoundcodes/amaquet/internal/protocol"
	"github.com/newfoundcodes/amaquet/internal/security"
	"github.com/newfoundcodes/amaquet/internal/server"
)

const version = "0.1.0"

func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)

	return "amaquet_admin_" + base64.RawURLEncoding.EncodeToString(b)
}

func main() {
	configPath := flag.String("config", "./amaquet.json", "path to JSON configuration")
	showVersion := flag.Bool("version", false, "print version")
	initConfig := flag.Bool("init-config", false, "write a default configuration and exit")

	flag.Parse()

	logger := log.New(os.Stdout)
	logger.SetLevel(log.InfoLevel)
	logger.SetReportTimestamp(true)
	logger.SetReportCaller(false)
	logger.SetPrefix("amaquet")

	if *showVersion {
		fmt.Println("Amaquet", version)
		return
	}

	if *initConfig {
		c := config.Default()
		c.Security.BootstrapToken = randomToken()

		if e := config.Save(*configPath, c); e != nil {
			logger.Fatal("Failed to write configuration file", "path", *configPath, "error", e)
		}

		logger.Info("Configuration file created", "path", *configPath)
		logger.Info("Bootstrap token generated — save this now, it will not be shown again", "token", c.Security.BootstrapToken)

		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", "path", *configPath, "error", err)
	}

	if err = os.MkdirAll(cfg.DataDir, 0700); err != nil {
		logger.Fatal("Cannot create data directory", "path", cfg.DataDir, "error", err)
	}

	if cfg.Security.BootstrapToken == "" {
		secretPath := filepath.Join(cfg.DataDir, "bootstrap-token")
		if b, readErr := os.ReadFile(secretPath); readErr == nil && strings.TrimSpace(string(b)) != "" {
			cfg.Security.BootstrapToken = strings.TrimSpace(string(b))
		} else {
			cfg.Security.BootstrapToken = randomToken()

			if err = os.WriteFile(secretPath, []byte(cfg.Security.BootstrapToken+"\n"), 0600); err != nil {
				logger.Fatal("Cannot persist bootstrap token", "path", secretPath, "error", err)
			}

			logger.Warn("No bootstrap token was set — a new one has been generated and saved",
				"path", secretPath,
				"action", "Read it from the protected file and configure a permanent admin API key",
			)
		}
	}

	manager := config.NewManager(*configPath, cfg)
	state, err := admin.NewState(filepath.Join(cfg.DataDir, "admin.json"), cfg.Security.BootstrapToken)

	if err != nil {
		logger.Fatal("Cannot load admin state", "error", err)
	}
	defer state.Close()

	engine := core.NewEngine()
	defer engine.Close()

	if err = engine.ConfigureMemory(cfg.Limits.MaxMemoryBytes, cfg.Limits.EvictionPolicy); err != nil {
		logger.Fatal("Invalid memory configuration", "max_memory_bytes", cfg.Limits.MaxMemoryBytes, "eviction_policy", cfg.Limits.EvictionPolicy, "error", err)
	}

	dispatcher := server.NewDispatcher(engine)
	defer dispatcher.Close()

	if err = dispatcher.ConfigureUploads(filepath.Join(cfg.DataDir, "uploads"), time.Duration(cfg.Limits.UploadTTLMS)*time.Millisecond, cfg.Limits.MaxPendingUploadBytes); err != nil {
		logger.Fatal("Invalid upload configuration", "error", err)
	}

	if err = dispatcher.SetCompression(compression.Config{Enabled: cfg.Compression.Enabled, Algorithm: cfg.Compression.Algorithm, MinBytes: cfg.Compression.MinBytes, MinSavingsPercent: cfg.Compression.MinSavingsPercent}); err != nil {
		logger.Fatal("Invalid compression configuration", "error", err)
	}

	logger.Info("In-memory compression configured", "enabled", cfg.Compression.Enabled, "algorithm", cfg.Compression.Algorithm, "min_bytes", cfg.Compression.MinBytes, "min_savings_percent", cfg.Compression.MinSavingsPercent)

	var aof *persistence.AOF
	if cfg.Persistence.AOFEnabled {
		replayed, e := persistence.Replay(cfg.Persistence.AOFPath, func(req protocol.Request) error {
			_, err := dispatcher.ExecuteReplay(context.Background(), req)
			return err
		})

		if e != nil {
			logger.Fatal("Cannot replay append-only file", "path", cfg.Persistence.AOFPath, "error", e)
		}

		dispatcher.DiscardIncompleteUploads()
		aof, e = persistence.Open(cfg.Persistence.AOFPath, cfg.Persistence.Fsync)

		if e != nil {
			logger.Fatal("Cannot open append-only file", "path", cfg.Persistence.AOFPath, "error", e)
		}

		defer aof.Close()
		dispatcher.SetMutationLog(aof)

		logger.Info("Append-only persistence enabled",
			"path", cfg.Persistence.AOFPath,
			"replayed_records", replayed,
			"fsync", cfg.Persistence.Fsync,
		)
	}

	var identity *security.Identity
	if cfg.Security.PublicKeyFile != "" && cfg.Security.PrivateKeyFile != "" {
		id, e := security.Load(cfg.Security.PublicKeyFile, cfg.Security.PrivateKeyFile)
		if e != nil {
			logger.Fatal("Cannot load protocol identity keys", "public", cfg.Security.PublicKeyFile, "private", cfg.Security.PrivateKeyFile, "error", e)
		}

		identity = &id
		logger.Info("Protocol identity loaded", "fingerprint", id.Fingerprint())
	}

	tcp := server.NewTCPServer(cfg.Protocol, dispatcher, state, identity, logger, cfg.Limits.MaxConnections, cfg.Limits.MaxPayloadBytes)
	tcp.SetRuntimeLimits(cfg.Limits.MaxInflightRequests, cfg.Limits.MaxInflightPerConnection, cfg.Limits.MaxInflightPayloadBytes, cfg.Admin.AuthFailuresPerMin)
	tcp.SetCommandTimeout(cfg.Limits.CommandTimeoutMS)

	adminAPI := &admin.HTTPServer{State: state, Engine: engine, Executor: dispatcher, Config: manager, MetricsRequireAuth: cfg.Admin.MetricsRequireAuth, MaxSessions: cfg.Admin.MaxSessions, AuthFailuresPerMin: cfg.Admin.AuthFailuresPerMin}
	state.OnActorRevoked(func(actor string) { adminAPI.InvalidateActorSessions(actor); tcp.InvalidateActor(actor) })

	if aof != nil {
		adminAPI.PersistenceHealth = aof.HealthError
		adminAPI.PersistenceCompact = func() error { return dispatcher.WithMutationBarrier(aof.Compact) }
		adminAPI.PersistenceCheckpoint = func(name string) (string, error) {
			if name == "" {
				name = "checkpoint-" + time.Now().UTC().Format("20060102T150405Z") + ".aof"
			}

			name = filepath.Base(name)
			dst := filepath.Join(cfg.DataDir, "backups", name)

			return dst, aof.Checkpoint(dst)
		}
	}

	httpSrv := &http.Server{Addr: net.JoinHostPort(cfg.Admin.Host, fmt.Sprintf("%d", cfg.Admin.Port)), Handler: adminAPI.Handler(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 90 * time.Second}
	errCh := make(chan error, 2)

	go func() { errCh <- tcp.ListenAndServe() }()

	go func() {
		logger.Info("Admin API listening", "address", httpSrv.Addr, "tls", cfg.Admin.TLS)
		if cfg.Admin.TLS {
			errCh <- httpSrv.ListenAndServeTLS(cfg.Admin.TLSCertFile, cfg.Admin.TLSKeyFile)
		} else {
			errCh <- httpSrv.ListenAndServe()
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case s := <-sig:
		logger.Info("Shutdown requested", "signal", s.String())
	case e := <-errCh:
		if e != nil && e != http.ErrServerClosed {
			logger.Error("Server stopped unexpectedly", "error", e)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = httpSrv.Shutdown(ctx)
	_ = tcp.Close()
}
