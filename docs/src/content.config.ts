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

import { defineCollection } from "astro:content"
import { glob } from "astro/loaders"
import { i18nLoader } from "@astrojs/starlight/loaders"
import { docsSchema, i18nSchema } from "@astrojs/starlight/schema"

const docs = defineCollection({
  loader: glob({
    base: ".",
    pattern: [
      "index.md",
      "404.md",
      "architecture.md",
      "protocol.md",
      "{concepts,data-types,development,getting-started,operations,protocol,reference}/**/*.md",
    ],
    generateId: ({ entry }) => entry.replace(/\.md$/, ""),
  }),
  schema: docsSchema(),
})

const i18n = defineCollection({
  loader: i18nLoader(),
  schema: i18nSchema(),
})

export const collections = { docs, i18n }
