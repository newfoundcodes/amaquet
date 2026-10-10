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
	"fmt"
	"net"
	"net/url"
)

type CIDRNetwork struct {
	CIDR    string `json:"cidr"`
	network *net.IPNet
}

func ParseCIDRNetwork(s string) (CIDRNetwork, error) {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		return CIDRNetwork{}, err
	}
	return CIDRNetwork{CIDR: n.String(), network: n}, nil
}
func (c CIDRNetwork) Contains(ip string) bool {
	p := net.ParseIP(ip)
	return p != nil && c.network != nil && c.network.Contains(p)
}

type URLValue struct {
	Raw    string `json:"raw"`
	parsed *url.URL
}

func ParseURL(s string) (URLValue, error) {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return URLValue{}, err
	}
	if u.Scheme == "" {
		return URLValue{}, fmt.Errorf("URL scheme is required")
	}
	return URLValue{Raw: u.String(), parsed: u}, nil
}
func (u URLValue) Scheme() string {
	if u.parsed == nil {
		p, _ := url.Parse(u.Raw)
		if p != nil {
			return p.Scheme
		}
		return ""
	}
	return u.parsed.Scheme
}
func (u URLValue) Host() string {
	if u.parsed == nil {
		p, _ := url.Parse(u.Raw)
		if p != nil {
			return p.Host
		}
		return ""
	}
	return u.parsed.Host
}
