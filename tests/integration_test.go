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

package tests

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/newfoundcodes/amaquet/internal/admin"
	"github.com/newfoundcodes/amaquet/internal/config"
	"github.com/newfoundcodes/amaquet/internal/core"
	"github.com/newfoundcodes/amaquet/internal/security"
	"github.com/newfoundcodes/amaquet/internal/server"
	"github.com/newfoundcodes/amaquet/pkg/amaquet"
)

func freePort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestAmaquetProtocolEndToEnd(t *testing.T) {
	engine := core.NewEngine()
	defer engine.Close()

	state, err := admin.NewState("", "bootstrap")
	if err != nil {
		t.Fatal(err)
	}

	_, key, err := state.CreateAPIKey("bootstrap", "integration", admin.RoleAdmin, nil)
	if err != nil {
		t.Fatal(err)
	}

	identity, err := security.Generate()
	if err != nil {
		t.Fatal(err)
	}

	port := freePort(t)
	cfg := config.ProtocolConfig{Host: "127.0.0.1", Port: port, RequireAuth: true}
	logger := log.New(io.Discard)
	srv := server.NewTCPServer(cfg, server.NewDispatcher(engine), state, &identity, logger, 32, 1<<20)
	done := make(chan error, 1)

	go func() {
		done <- srv.ListenAndServe()
	}()

	defer func() {
		_ = srv.Close()
		<-done
	}()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(2 * time.Second)

	for {
		c, e := net.DialTimeout("tcp", addr, 20*time.Millisecond)
		if e == nil {
			_ = c.Close()
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("server did not start: %v", e)
		}

		time.Sleep(10 * time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	uri := fmt.Sprintf("amaquet://127.0.0.1:%d", port)
	client, err := amaquet.DialWithOptions(ctx, uri, amaquet.DialOptions{APIKey: key})

	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err = client.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	info, err := client.Hello(ctx, "test-nonce")
	if err != nil {
		t.Fatal(err)
	}

	if !info.VerifyNonce("test-nonce") {
		t.Fatal("server identity nonce did not verify")
	}

	if err = client.Set(ctx, "counter", "integer", int64(41), 0); err != nil {
		t.Fatal(err)
	}

	got, err := client.Get(ctx, "counter")
	if err != nil {
		t.Fatal(err)
	}

	if got["type"] != "integer" || got["value"].(float64) != 41 {
		t.Fatalf("unexpected GET: %#v", got)
	}

	var n int64
	if err = client.Op(ctx, "counter", "ADD", map[string]any{"delta": 1}, &n); err != nil {
		t.Fatal(err)
	}

	if n != 42 {
		t.Fatalf("counter = %d", n)
	}

	if err = client.Create(ctx, "names", "list", nil); err != nil {
		t.Fatal(err)
	}

	var listLen int
	v := map[string]any{
		"type":  "utf8_string",
		"value": "Nathanne",
	}

	if err = client.Op(ctx, "names", "PUSH_BACK", map[string]any{
		"value": v,
	}, &listLen); err != nil {
		t.Fatal(err)
	}

	if listLen != 1 {
		t.Fatalf("list length = %d", listLen)
	}

	if err = client.Create(ctx, "events", "pubsub", nil); err != nil {
		t.Fatal(err)
	}

	sub, err := client.Subscribe(ctx, "events", "updates", 8)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Close()

	var subscribers int
	if err = client.Op(ctx, "events", "PUBLISH", map[string]any{
		"channel": "updates",
		"value":   v,
	}, &subscribers); err != nil {
		t.Fatal(err)
	}

	if subscribers != 1 {
		t.Fatalf("subscribers = %d", subscribers)
	}

	select {
	case msg := <-sub.C:
		if msg.Channel != "updates" {
			t.Fatalf("unexpected channel: %s", msg.Channel)
		}

	case <-time.After(time.Second):
		t.Fatal("timed out waiting for pub/sub event")
	}
}
