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

import "testing"

func TestProductionEnvironmentOverrides(t *testing.T) {
	t.Setenv("AMAQUET_AOF_ENABLED", "true")
	t.Setenv("AMAQUET_AOF_PATH", "/tmp/amaquet-test.aof")
	t.Setenv("AMAQUET_AOF_FSYNC", "always")
	t.Setenv("AMAQUET_MAX_MEMORY_BYTES", "1048576")
	t.Setenv("AMAQUET_EVICTION_POLICY", "allkeys-lru")
	t.Setenv("AMAQUET_MAX_INFLIGHT_REQUESTS", "123")
	t.Setenv("AMAQUET_MAX_INFLIGHT_PER_CONNECTION", "7")
	t.Setenv("AMAQUET_MAX_INFLIGHT_PAYLOAD_BYTES", "67108864")
	t.Setenv("AMAQUET_MAX_PENDING_UPLOAD_BYTES", "33554432")
	t.Setenv("AMAQUET_UPLOAD_TTL_MS", "30000")

	c := Default()
	applyEnv(&c)

	if !c.Persistence.AOFEnabled ||
		c.Persistence.AOFPath != "/tmp/amaquet-test.aof" ||
		c.Persistence.Fsync != "always" {
		t.Fatalf("AOF env overrides not applied: %+v", c.Persistence)
	}

	if c.Limits.MaxMemoryBytes != 1048576 || c.Limits.EvictionPolicy != "allkeys-lru" {
		t.Fatalf("memory env overrides not applied: %+v", c.Limits)
	}

	if c.Limits.MaxInflightRequests != 123 ||
		c.Limits.MaxInflightPerConnection != 7 ||
		c.Limits.MaxPendingUploadBytes != 33554432 ||
		c.Limits.UploadTTLMS != 30000 {
		t.Fatalf("runtime limit env overrides not applied: %+v", c.Limits)
	}
}
