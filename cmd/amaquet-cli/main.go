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
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/newfoundcodes/amaquet/pkg/amaquet"
)

func main() {
	uri := flag.String("uri", "amaquet://127.0.0.1:13378", "Amaquet URI")
	apiKey := flag.String("api-key", "", "API key (preferred over embedding a secret in the URI)")
	allowURISecrets := flag.Bool("allow-uri-secrets", false, "allow legacy API keys in URI user-info/query parameters")
	timeout := flag.Duration("timeout", 30*time.Second, "request timeout")

	flag.Parse()

	if flag.NArg() < 1 {
		log.Error("No command specified",
			"usage", "amaquet-cli [-uri amaquet://...] COMMAND ['{\"arg\":...}']")
		os.Exit(2)
	}

	command := strings.ToUpper(flag.Arg(0))
	args := map[string]any{}

	if flag.NArg() > 1 {
		raw := flag.Arg(1)

		var data []byte
		var readErr error

		switch {
		case raw == "-":
			data, readErr = os.ReadFile("/dev/stdin")
		case strings.HasPrefix(raw, "@"):
			if len(raw) == 1 {
				readErr = fmt.Errorf("@file argument requires a path")
			} else {
				data, readErr = os.ReadFile(raw[1:])
			}
		default:
			data = []byte(raw)
		}

		if readErr != nil {
			log.Error("Cannot read argument JSON", "error", readErr)
			os.Exit(2)
		}

		dec := json.NewDecoder(strings.NewReader(string(data)))
		dec.UseNumber()

		if err := dec.Decode(&args); err != nil {
			log.Error("Invalid argument JSON", "error", err)
			os.Exit(2)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	c, err := amaquet.DialWithOptions(ctx, *uri, amaquet.DialOptions{APIKey: *apiKey, Timeout: *timeout, AllowURISecrets: *allowURISecrets})
	if err != nil {
		fatal(err)
	}
	defer c.Close()

	var out json.RawMessage
	if err = c.Command(ctx, command, args, &out); err != nil {
		fatal(err)
	}

	var pretty bytes.Buffer
	if json.Indent(&pretty, out, "", "  ") == nil {
		fmt.Println(pretty.String())
	} else {
		fmt.Println(string(out))
	}
}

func fatal(err error) {
	log.Fatal("Command failed", "error", err)
}
