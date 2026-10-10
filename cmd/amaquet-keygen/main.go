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
	"github.com/newfoundcodes/amaquet/internal/security"
)

func main() {
	pub := flag.String("public", "amaquet-public.pem", "public key output")
	priv := flag.String("private", "amaquet-private.pem", "private key output")

	flag.Parse()

	logger := log.New(os.Stdout)
	logger.SetPrefix("amaquet-keygen")
	logger.SetReportTimestamp(false)

	id, e := security.Generate()
	if e != nil {
		logger.Fatal("Failed to generate key pair", "error", e)
	}

	if e = security.Save(id, *pub, *priv); e != nil {
		logger.Fatal("Failed to save key pair", "public", *pub, "private", *priv, "error", e)
	}

	logger.Info("Key pair generated successfully",
		"public_key", *pub,
		"private_key", *priv,
		"fingerprint", id.Fingerprint(),
	)
}
