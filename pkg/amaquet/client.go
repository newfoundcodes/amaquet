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
	"crypto/ed25519"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/newfoundcodes/amaquet/internal/protocol"
)

type Client struct {
	endpoint  protocol.Endpoint
	conn      net.Conn
	writeMu   sync.Mutex
	next      atomic.Uint64
	pendingMu sync.Mutex
	subsMu    sync.Mutex
	closeOnce sync.Once
	Timeout   time.Duration
	TLSConfig *tls.Config
	pending   map[uint64]chan protocol.Frame
	subs      map[uint64]chan PubSubMessage
	closed    chan struct{}
}

type ServerInfo struct {
	Name                string   `json:"name"`
	Version             string   `json:"version"`
	Protocol            int      `json:"protocol"`
	SelectedProtocol    int      `json:"selected_protocol,omitempty"`
	Capabilities        []string `json:"capabilities,omitempty"`
	AuthRequired        bool     `json:"auth_required"`
	IdentityPublicKey   string   `json:"identity_public_key,omitempty"`
	IdentityFingerprint string   `json:"identity_fingerprint,omitempty"`
	NonceSignature      string   `json:"nonce_signature,omitempty"`
}

type PubSubMessage struct {
	Channel     string    `json:"channel"`
	Payload     any       `json:"payload"`
	PublishedAt time.Time `json:"published_at"`
}

type Subscription struct {
	C     <-chan PubSubMessage
	close func() error
}

type DialOptions struct {
	APIKey          string
	TLSConfig       *tls.Config
	Timeout         time.Duration
	AllowURISecrets bool
}

func (s *Subscription) Close() error {
	if s == nil || s.close == nil {
		return nil
	}

	return s.close()
}

func Dial(ctx context.Context, uri string) (*Client, error) {
	return DialWithOptions(ctx, uri, DialOptions{})
}

func DialWithOptions(ctx context.Context, uri string, opts DialOptions) (*Client, error) {
	ep, err := protocol.ParseEndpoint(uri)
	if err != nil {
		return nil, err
	}

	if ep.APIKey != "" && !opts.AllowURISecrets && opts.APIKey == "" {
		return nil, fmt.Errorf("API keys in amaquet:// URIs are disabled; pass DialOptions.APIKey")
	}

	apiKey := opts.APIKey
	if apiKey == "" {
		apiKey = ep.APIKey
	}

	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", ep.Address())

	if err != nil {
		return nil, err
	}

	if ep.TLS {
		cfg := opts.TLSConfig
		if cfg == nil {
			cfg = &tls.Config{ServerName: ep.Host, MinVersion: tls.VersionTLS12}
		} else {
			cfg = cfg.Clone()
			if cfg.ServerName == "" {
				cfg.ServerName = ep.Host
			}

			if cfg.MinVersion == 0 {
				cfg.MinVersion = tls.VersionTLS12
			}
		}

		tc := tls.Client(conn, cfg)
		if err = tc.HandshakeContext(ctx); err != nil {
			conn.Close()
			return nil, err
		}

		conn = tc
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	c := &Client{
		endpoint:  ep,
		conn:      conn,
		Timeout:   timeout,
		TLSConfig: opts.TLSConfig,
		pending:   map[uint64]chan protocol.Frame{},
		subs:      map[uint64]chan PubSubMessage{},
		closed:    make(chan struct{}),
	}
	go c.readLoop()

	var info ServerInfo
	if err = c.request(ctx, protocol.OpHello, map[string]any{
		"min_protocol": int(protocol.Version),
		"max_protocol": int(protocol.Version),
	}, &info, nil); err != nil {
		c.Close()
		return nil, err
	}

	if info.SelectedProtocol != 0 && info.SelectedProtocol != int(protocol.Version) {
		c.Close()
		return nil, fmt.Errorf("server selected unsupported protocol %d", info.SelectedProtocol)
	}

	if apiKey != "" {
		var out any
		if err = c.request(ctx, protocol.OpAuth, map[string]any{
			"api_key": apiKey,
		}, &out, nil); err != nil {
			c.Close()
			return nil, err
		}
	}

	return c, nil
}

func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closed)
		err = c.conn.Close()
	})

	return err
}

func (c *Client) readLoop() {
	r := bufio.NewReader(c.conn)
	defer func() {
		c.closeOnce.Do(func() {
			close(c.closed)
			_ = c.conn.Close()
		})
		c.pendingMu.Lock()

		for id, ch := range c.pending {
			close(ch)
			delete(c.pending, id)
		}

		c.pendingMu.Unlock()
		c.subsMu.Lock()

		for id, ch := range c.subs {
			close(ch)
			delete(c.subs, id)
		}

		c.subsMu.Unlock()
	}()

	for {
		f, err := protocol.ReadBufferedFrame(r)
		if err != nil {
			return
		}

		if f.Kind == protocol.KindEvent {
			var m PubSubMessage
			if json.Unmarshal(f.Payload, &m) == nil {
				c.subsMu.Lock()
				ch := c.subs[f.RequestID]

				if ch != nil {
					select {
					case ch <- m:
					default:
					}
				}
				c.subsMu.Unlock()
			}
			continue
		}
		c.pendingMu.Lock()

		ch := c.pending[f.RequestID]
		if ch != nil {
			delete(c.pending, f.RequestID)
			ch <- f

			close(ch)
		}

		c.pendingMu.Unlock()
	}
}

func (c *Client) sendCancel(target uint64) {
	id := c.next.Add(1)
	b, _ := json.Marshal(map[string]any{"request_id": target})

	go func() {
		select {
		case <-c.closed:
			return
		default:
		}

		c.writeMu.Lock()
		defer c.writeMu.Unlock()

		_ = protocol.WriteFrame(c.conn, protocol.Frame{
			Kind:      protocol.KindRequest,
			Opcode:    protocol.OpCancel,
			RequestID: id, Payload: b,
		})
	}()
}

func (c *Client) request(ctx context.Context, op protocol.Opcode, payload any, out any, eventCh chan PubSubMessage) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	id := c.next.Add(1)
	ch := make(chan protocol.Frame, 1)
	c.pendingMu.Lock()

	c.pending[id] = ch
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()
	}()

	if eventCh != nil {
		c.subsMu.Lock()
		c.subs[id] = eventCh

		c.subsMu.Unlock()
	}
	c.writeMu.Lock()

	err = protocol.WriteFrame(c.conn, protocol.Frame{
		Kind:      protocol.KindRequest,
		Opcode:    op,
		RequestID: id,
		Payload:   b,
	})
	c.writeMu.Unlock()

	if err != nil {
		c.pendingMu.Lock()
		delete(c.pending, id)

		c.pendingMu.Unlock()
		if eventCh != nil {
			c.subsMu.Lock()
			delete(c.subs, id)

			c.subsMu.Unlock()
		}

		return err
	}

	wait := c.Timeout
	if wait <= 0 {
		wait = 30 * time.Second
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		c.sendCancel(id)
		if eventCh != nil {
			c.subsMu.Lock()
			delete(c.subs, id)

			c.subsMu.Unlock()
		}
		return ctx.Err()

	case <-timer.C:
		c.sendCancel(id)
		if eventCh != nil {
			c.subsMu.Lock()
			delete(c.subs, id)

			c.subsMu.Unlock()
		}
		return context.DeadlineExceeded

	case <-c.closed:
		if eventCh != nil {
			c.subsMu.Lock()
			delete(c.subs, id)
			c.subsMu.Unlock()
		}
		return net.ErrClosed

	case f, ok := <-ch:
		if !ok {
			return net.ErrClosed
		}

		var resp struct {
			OK     bool            `json:"ok"`
			Result json.RawMessage `json:"result"`
			Error  *protocol.Error `json:"error"`
		}

		if err = json.Unmarshal(f.Payload, &resp); err != nil {
			return err
		}

		if !resp.OK {
			if eventCh != nil {
				c.subsMu.Lock()
				delete(c.subs, id)

				c.subsMu.Unlock()
			}

			if resp.Error == nil {
				return fmt.Errorf("request failed")
			}

			return fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message)
		}

		if out != nil {
			if raw, ok := out.(*json.RawMessage); ok {
				*raw = append((*raw)[:0], resp.Result...)
				return nil
			}

			return json.Unmarshal(resp.Result, out)
		}

		return nil
	}
}

func (c *Client) Command(ctx context.Context, command string, args map[string]any, out any) error {
	return c.request(ctx, protocol.OpCommand, map[string]any{
		"command": command,
		"args":    args,
	}, out, nil)
}

func (c *Client) Hello(ctx context.Context, nonce string) (ServerInfo, error) {
	var out ServerInfo
	err := c.request(ctx, protocol.OpHello, map[string]any{
		"nonce":        nonce,
		"min_protocol": int(protocol.Version),
		"max_protocol": int(protocol.Version),
	}, &out, nil)

	return out, err
}

func (i ServerInfo) VerifyNonce(nonce string) bool {
	if i.IdentityPublicKey == "" || i.NonceSignature == "" {
		return false
	}

	pub, e := base64.RawStdEncoding.DecodeString(i.IdentityPublicKey)
	if e != nil || len(pub) != ed25519.PublicKeySize {
		return false
	}

	sig, e := base64.RawStdEncoding.DecodeString(i.NonceSignature)
	return e == nil && ed25519.Verify(ed25519.PublicKey(pub), []byte(nonce), sig)
}

func (c *Client) Ping(ctx context.Context) error {
	var out string
	return c.Command(ctx, "PING", nil, &out)
}

func (c *Client) Set(ctx context.Context, key, typeName string, value any, ttl time.Duration) error {
	return c.Command(ctx, "SET", map[string]any{
		"key":    key,
		"type":   typeName,
		"value":  value,
		"ttl_ms": ttl.Milliseconds(),
	}, nil)
}

func (c *Client) Get(ctx context.Context, key string) (map[string]any, error) {
	var out map[string]any
	err := c.Command(ctx, "GET", map[string]any{"key": key}, &out)

	return out, err
}

func (c *Client) Delete(ctx context.Context, keys ...string) (int, error) {
	var n int
	err := c.Command(ctx, "DEL", map[string]any{"keys": keys}, &n)

	return n, err
}

func (c *Client) Create(ctx context.Context, key, typeName string, options map[string]any) error {
	return c.Command(ctx, "CREATE", map[string]any{"key": key, "type": typeName, "options": options}, nil)
}

func (c *Client) Op(ctx context.Context, key, operation string, args map[string]any, out any) error {
	return c.Command(ctx, "OP", map[string]any{"key": key, "operation": operation, "args": args}, out)
}

func (c *Client) Subscribe(ctx context.Context, key, channel string, buffer int) (*Subscription, error) {
	if buffer < 1 {
		buffer = 64
	}

	events := make(chan PubSubMessage, buffer)
	id := c.next.Add(1)
	ch := make(chan protocol.Frame, 1)

	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	c.subsMu.Lock()
	c.subs[id] = events
	c.subsMu.Unlock()

	cleanup := func() {
		c.pendingMu.Lock()
		delete(c.pending, id)
		c.pendingMu.Unlock()

		c.subsMu.Lock()
		if x := c.subs[id]; x != nil {
			delete(c.subs, id)
			close(x)
		}

		c.subsMu.Unlock()
	}

	b, _ := json.Marshal(map[string]any{
		"key":     key,
		"channel": channel,
		"buffer":  buffer,
	})
	c.writeMu.Lock()

	err := protocol.WriteFrame(c.conn, protocol.Frame{
		Kind:      protocol.KindRequest,
		Opcode:    protocol.OpSubscribe,
		RequestID: id,
		Payload:   b,
	})
	c.writeMu.Unlock()

	if err != nil {
		cleanup()
		return nil, err
	}

	select {
	case <-ctx.Done():
		c.sendCancel(id)
		c.pendingMu.Lock()
		delete(c.pending, id)

		c.pendingMu.Unlock()
		c.subsMu.Lock()
		delete(c.subs, id)

		c.subsMu.Unlock()
		return nil, ctx.Err()

	case <-c.closed:
		c.pendingMu.Lock()
		delete(c.pending, id)

		c.pendingMu.Unlock()
		c.subsMu.Lock()
		delete(c.subs, id)

		c.subsMu.Unlock()
		return nil, net.ErrClosed

	case f := <-ch:
		var resp protocol.Response
		if e := json.Unmarshal(f.Payload, &resp); e != nil {
			cleanup()
			return nil, e
		}

		if !resp.OK {
			cleanup()
			return nil, fmt.Errorf("subscribe failed: %v", resp.Error)
		}
	}

	sub := &Subscription{C: events}
	sub.close = func() error {
		var out any
		err := c.request(context.Background(), protocol.OpUnsubscribe, map[string]any{
			"key":     key,
			"channel": channel,
		}, &out, nil)
		c.subsMu.Lock()

		if x := c.subs[id]; x != nil {
			delete(c.subs, id)
			close(x)
		}

		c.subsMu.Unlock()
		return err
	}

	return sub, nil
}

func (c *Client) BlobRead(ctx context.Context, key string, offset int64, length int) ([]byte, int64, error) {
	var out struct {
		Data string `json:"data"`
		Size int64  `json:"size"`
	}

	if err := c.Command(ctx, "BLOB_READ", map[string]any{
		"key":    key,
		"offset": offset,
		"length": length,
	}, &out); err != nil {
		return nil, 0, err
	}

	b, err := base64.StdEncoding.DecodeString(out.Data)
	return b, out.Size, err
}

func (c *Client) UploadBlob(ctx context.Context, key, uploadID string, r io.Reader, size int64, chunkSize int) error {
	if size < 0 {
		return fmt.Errorf("size must be non-negative")
	}

	if uploadID == "" {
		uploadID = fmt.Sprintf("client-%d", c.next.Add(1))
	}

	if chunkSize <= 0 {
		chunkSize = 1 << 20
	}

	if chunkSize > 32<<20 {
		chunkSize = 32 << 20
	}

	if err := c.Command(ctx, "BLOB_BEGIN", map[string]any{
		"key":        key,
		"upload_id":  uploadID,
		"total_size": size,
	}, nil); err != nil {
		return err
	}

	abort := true
	defer func() {
		if abort {
			_ = c.Command(context.Background(), "BLOB_ABORT", map[string]any{
				"upload_id": uploadID,
			}, nil)
		}
	}()

	buf := make([]byte, chunkSize)

	var off int64
	for off < size {
		want := int64(len(buf))
		if size-off < want {
			want = size - off
		}

		n, err := io.ReadFull(r, buf[:want])
		if err != nil {
			return err
		}

		if n == 0 {
			return io.ErrUnexpectedEOF
		}

		if err = c.Command(ctx, "BLOB_CHUNK", map[string]any{
			"upload_id": uploadID,
			"offset":    off,
			"data":      base64.StdEncoding.EncodeToString(buf[:n]),
		}, nil); err != nil {
			return err
		}

		off += int64(n)
	}

	if err := c.Command(ctx, "BLOB_COMMIT", map[string]any{
		"upload_id": uploadID,
	}, nil); err != nil {
		return err
	}

	abort = false
	return nil
}
