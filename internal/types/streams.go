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

package types

import (
	"context"
	"fmt"
	"github.com/newfoundcodes/amaquet/internal/core"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type StreamID struct {
	Millis   int64  `json:"millis"`
	Sequence uint64 `json:"sequence"`
}

func (id StreamID) String() string {
	return strconv.FormatInt(id.Millis, 10) + "-" + strconv.FormatUint(id.Sequence, 10)
}
func ParseStreamID(s string) (StreamID, error) {
	p := strings.Split(s, "-")
	if len(p) != 2 {
		return StreamID{}, fmt.Errorf("invalid stream id")
	}
	ms, e := strconv.ParseInt(p[0], 10, 64)
	if e != nil {
		return StreamID{}, e
	}
	seq, e := strconv.ParseUint(p[1], 10, 64)
	return StreamID{ms, seq}, e
}
func (a StreamID) Less(b StreamID) bool {
	return a.Millis < b.Millis || a.Millis == b.Millis && a.Sequence < b.Sequence
}

type StreamEntry struct {
	ID     StreamID              `json:"id"`
	Fields map[string]core.Value `json:"fields"`
}
type Stream struct {
	mu      sync.RWMutex
	entries []StreamEntry
	last    StreamID
	groups  map[string]*ConsumerGroup
	notify  chan struct{}
}

func NewStream() *Stream {
	return &Stream{groups: map[string]*ConsumerGroup{}, notify: make(chan struct{}, 1)}
}
func (s *Stream) NextIDAt(ms int64) StreamID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ms <= s.last.Millis {
		return StreamID{s.last.Millis, s.last.Sequence + 1}
	}
	return StreamID{ms, 0}
}
func (s *Stream) Add(fields map[string]core.Value, id *StreamID) (StreamID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var next StreamID
	if id == nil {
		now := time.Now().UnixMilli()
		if now == s.last.Millis {
			next = StreamID{now, s.last.Sequence + 1}
		} else if now < s.last.Millis {
			next = StreamID{s.last.Millis, s.last.Sequence + 1}
		} else {
			next = StreamID{now, 0}
		}
	} else {
		next = *id
		if !s.last.Less(next) && s.last != (StreamID{}) {
			return StreamID{}, fmt.Errorf("stream ID must increase")
		}
	}
	cp := make(map[string]core.Value, len(fields))
	for k, v := range fields {
		cp[k] = v
	}
	s.entries = append(s.entries, StreamEntry{ID: next, Fields: cp})
	s.last = next
	select {
	case s.notify <- struct{}{}:
	default:
	}
	return next, nil
}
func (s *Stream) Range(start, end *StreamID, count int) []StreamEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if count <= 0 {
		count = len(s.entries)
	}
	out := make([]StreamEntry, 0)
	for _, e := range s.entries {
		if start != nil && e.ID.Less(*start) {
			continue
		}
		if end != nil && end.Less(e.ID) {
			continue
		}
		out = append(out, e)
		if len(out) >= count {
			break
		}
	}
	return out
}
func (s *Stream) ReadAfter(ctx context.Context, last StreamID, count int, block bool) ([]StreamEntry, error) {
	for {
		entries := s.Range(&last, nil, count+1)
		out := entries[:0]
		for _, e := range entries {
			if last.Less(e.ID) {
				out = append(out, e)
			}
		}
		if len(out) > 0 || !block {
			return out, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-s.notify:
		}
	}
}
func (s *Stream) Trim(maxLen int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if maxLen < 0 {
		maxLen = 0
	}
	if len(s.entries) <= maxLen {
		return 0
	}
	n := len(s.entries) - maxLen
	s.entries = append([]StreamEntry(nil), s.entries[n:]...)
	return n
}
func (s *Stream) Len() int { s.mu.RLock(); defer s.mu.RUnlock(); return len(s.entries) }

type PendingMessage struct {
	Entry         StreamEntry `json:"entry"`
	Consumer      string      `json:"consumer"`
	Deliveries    int         `json:"deliveries"`
	LastDelivered time.Time   `json:"last_delivered"`
}
type ConsumerGroup struct {
	mu            sync.Mutex
	Name          string                    `json:"name"`
	LastDelivered StreamID                  `json:"last_delivered"`
	Pending       map[string]PendingMessage `json:"pending"`
}

func NewConsumerGroup(name string, start StreamID) *ConsumerGroup {
	return &ConsumerGroup{Name: name, LastDelivered: start, Pending: map[string]PendingMessage{}}
}
func (s *Stream) CreateGroup(name string, start StreamID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.groups[name]; ok {
		return fmt.Errorf("group exists")
	}
	s.groups[name] = NewConsumerGroup(name, start)
	return nil
}
func (s *Stream) DestroyGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.groups[name]; !ok {
		return false
	}
	delete(s.groups, name)
	return true
}
func (s *Stream) ReadGroup(group, consumer string, count int) ([]StreamEntry, error) {
	s.mu.RLock()
	g, ok := s.groups[group]
	if !ok {
		s.mu.RUnlock()
		return nil, fmt.Errorf("group not found")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	last := g.LastDelivered
	entries := make([]StreamEntry, 0)
	for _, e := range s.entries {
		if last.Less(e.ID) {
			entries = append(entries, e)
			if count > 0 && len(entries) >= count {
				break
			}
		}
	}
	s.mu.RUnlock()
	now := time.Now().UTC()
	for _, e := range entries {
		p := g.Pending[e.ID.String()]
		p.Entry = e
		p.Consumer = consumer
		p.Deliveries++
		p.LastDelivered = now
		g.Pending[e.ID.String()] = p
		g.LastDelivered = e.ID
	}
	return entries, nil
}
func (s *Stream) Ack(group string, ids ...StreamID) (int, error) {
	s.mu.RLock()
	g, ok := s.groups[group]
	s.mu.RUnlock()
	if !ok {
		return 0, fmt.Errorf("group not found")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	n := 0
	for _, id := range ids {
		if _, ok := g.Pending[id.String()]; ok {
			delete(g.Pending, id.String())
			n++
		}
	}
	return n, nil
}
func (s *Stream) Pending(group string) ([]PendingMessage, error) {
	s.mu.RLock()
	g, ok := s.groups[group]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("group not found")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]PendingMessage, 0, len(g.Pending))
	for _, p := range g.Pending {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Entry.ID.Less(out[j].Entry.ID) })
	return out, nil
}

type PubSubMessage struct {
	Channel     string     `json:"channel"`
	Payload     core.Value `json:"payload"`
	PublishedAt time.Time  `json:"published_at"`
}
type subscription struct{ ch chan PubSubMessage }
type PubSub struct {
	mu        sync.RWMutex
	channels  map[string]map[*subscription]struct{}
	published atomic.Uint64
	delivered atomic.Uint64
	dropped   atomic.Uint64
}

func NewPubSub() *PubSub { return &PubSub{channels: map[string]map[*subscription]struct{}{}} }

type Subscription struct {
	C      <-chan PubSubMessage
	cancel func()
}

func (s Subscription) Close() {
	if s.cancel != nil {
		s.cancel()
	}
}
func (p *PubSub) Subscribe(channel string, buffer int) Subscription {
	if buffer < 1 {
		buffer = 16
	}
	sub := &subscription{ch: make(chan PubSubMessage, buffer)}
	p.mu.Lock()
	if p.channels[channel] == nil {
		p.channels[channel] = map[*subscription]struct{}{}
	}
	p.channels[channel][sub] = struct{}{}
	p.mu.Unlock()
	return Subscription{C: sub.ch, cancel: func() {
		p.mu.Lock()
		if set := p.channels[channel]; set != nil {
			if _, ok := set[sub]; ok {
				delete(set, sub)
				close(sub.ch)
				if len(set) == 0 {
					delete(p.channels, channel)
				}
			}
		}
		p.mu.Unlock()
	}}
}
func (p *PubSub) Publish(channel string, v core.Value) int {
	msg := PubSubMessage{Channel: channel, Payload: v, PublishedAt: time.Now().UTC()}
	p.published.Add(1)
	p.mu.RLock()
	defer p.mu.RUnlock()
	n := 0
	for sub := range p.channels[channel] {
		select {
		case sub.ch <- msg:
			n++
			p.delivered.Add(1)
		default:
			p.dropped.Add(1)
		}
	}
	return n
}
func (p *PubSub) Stats() map[string]uint64 {
	return map[string]uint64{"published": p.published.Load(), "delivered": p.delivered.Load(), "dropped": p.dropped.Load()}
}

type TopicMessage struct {
	Offset    uint64     `json:"offset"`
	Key       string     `json:"key,omitempty"`
	Value     core.Value `json:"value"`
	Timestamp time.Time  `json:"timestamp"`
}
type PersistentTopic struct {
	mu          sync.RWMutex
	messages    []TopicMessage
	next        uint64
	MaxMessages int
}

func NewPersistentTopic(max int) *PersistentTopic { return &PersistentTopic{MaxMessages: max} }
func (t *PersistentTopic) Publish(key string, v core.Value) uint64 {
	return t.PublishAt(key, v, time.Now().UTC())
}
func (t *PersistentTopic) PublishAt(key string, v core.Value, at time.Time) uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	off := t.next
	t.next++
	t.messages = append(t.messages, TopicMessage{Offset: off, Key: key, Value: v, Timestamp: at.UTC()})
	if t.MaxMessages > 0 && len(t.messages) > t.MaxMessages {
		drop := len(t.messages) - t.MaxMessages
		t.messages = append([]TopicMessage(nil), t.messages[drop:]...)
	}
	return off
}
func (t *PersistentTopic) Read(from uint64, count int) []TopicMessage {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]TopicMessage, 0)
	for _, m := range t.messages {
		if m.Offset >= from {
			out = append(out, m)
			if count > 0 && len(out) >= count {
				break
			}
		}
	}
	return out
}

type Event struct {
	Sequence    uint64     `json:"sequence"`
	Type        string     `json:"type"`
	AggregateID string     `json:"aggregate_id,omitempty"`
	Payload     core.Value `json:"payload"`
	At          time.Time  `json:"at"`
}
type EventLog struct {
	mu     sync.RWMutex
	events []Event
	next   uint64
}

func NewEventLog() *EventLog { return &EventLog{next: 1} }
func (l *EventLog) Append(eventType, aggregateID string, payload core.Value) Event {
	return l.AppendAt(eventType, aggregateID, payload, time.Now().UTC())
}
func (l *EventLog) AppendAt(eventType, aggregateID string, payload core.Value, at time.Time) Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := Event{Sequence: l.next, Type: eventType, AggregateID: aggregateID, Payload: payload, At: at.UTC()}
	l.next++
	l.events = append(l.events, e)
	return e
}
func (l *EventLog) Read(from uint64, count int) []Event {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Event, 0)
	for _, e := range l.events {
		if e.Sequence >= from {
			out = append(out, e)
			if count > 0 && len(out) >= count {
				break
			}
		}
	}
	return out
}
func (l *EventLog) Len() int { l.mu.RLock(); defer l.mu.RUnlock(); return len(l.events) }
