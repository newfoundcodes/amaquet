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

package protocol

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
)

const DefaultPort = 13378

type Endpoint struct {
	TLS      bool
	Host     string
	Port     int
	Database string
	APIKey   string
}

func ParseEndpoint(raw string) (Endpoint, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Endpoint{}, err
	}
	if u.Scheme != "amaquet" && u.Scheme != "amaquets" {
		return Endpoint{}, fmt.Errorf("scheme must be amaquet or amaquets")
	}
	host := u.Hostname()
	if host == "" {
		return Endpoint{}, fmt.Errorf("host is required")
	}
	port := DefaultPort
	if u.Port() != "" {
		p, e := strconv.Atoi(u.Port())
		if e != nil || p < 1 || p > 65535 {
			return Endpoint{}, fmt.Errorf("invalid port")
		}
		port = p
	}
	db := u.Path
	if len(db) > 0 && db[0] == '/' {
		db = db[1:]
	}
	if db != "" {
		return Endpoint{}, fmt.Errorf("logical database paths are not supported; omit /%s", db)
	}
	apiKey := ""
	if u.User != nil {
		if p, ok := u.User.Password(); ok {
			apiKey = p
		} else {
			apiKey = u.User.Username()
		}
	}
	if q := u.Query().Get("api_key"); q != "" {
		apiKey = q
	}
	return Endpoint{TLS: u.Scheme == "amaquets", Host: host, Port: port, Database: db, APIKey: apiKey}, nil
}
func (e Endpoint) Address() string { return net.JoinHostPort(e.Host, strconv.Itoa(e.Port)) }
