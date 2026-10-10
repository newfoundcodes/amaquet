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

package amaquet

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/newfoundcodes/amaquet/internal/protocol"
)

func TestTimedOutRequestDoesNotLeakPendingEntry(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()

	c := &Client{
		conn:    clientConn,
		Timeout: 25 * time.Millisecond,
		pending: map[uint64]chan protocol.Frame{},
		subs:    map[uint64]chan PubSubMessage{},
		closed:  make(chan struct{}),
	}
	defer c.Close()

	go func() {
		r := bufio.NewReader(serverConn)
		_, _ = protocol.ReadBufferedFrame(r)

		time.Sleep(100 * time.Millisecond)
	}()

	err := c.request(context.Background(), protocol.OpCommand, map[string]any{
		"command": "PING",
	}, nil, nil)

	if err == nil {
		t.Fatal("expected timeout")
	}
	c.pendingMu.Lock()

	n := len(c.pending)
	c.pendingMu.Unlock()

	if n != 0 {
		t.Fatalf("pending leak: %d", n)
	}
}

func TestDialRejectsURIEmbeddedSecretByDefault(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := Dial(ctx, "amaquet://127.0.0.1:1?api_key=secret")
	if err == nil || !strings.Contains(err.Error(), "API keys in amaquet:// URIs are disabled") {
		t.Fatalf("expected URI-secret rejection, got %v", err)
	}
}
