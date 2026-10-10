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
	"flag"
	"os"

	"github.com/charmbracelet/log"
	"github.com/newfoundcodes/amaquet/internal/persistence"
)

func main() {
	src := flag.String("source", "", "verified checkpoint/AOF source")
	dst := flag.String("target", "", "target AOF path (server must be stopped)")

	flag.Parse()

	logger := log.New(os.Stdout)
	logger.SetPrefix("amaquet-restore")
	logger.SetReportTimestamp(false)

	if *src == "" || *dst == "" {
		logger.Fatal("Both -source and -target flags are required")
	}

	if err := persistence.RestoreCheckpoint(*src, *dst); err != nil {
		logger.Fatal("Restore failed", "source", *src, "target", *dst, "error", err)
	}

	logger.Info("Checkpoint restored successfully", "source", *src, "target", *dst)
}
