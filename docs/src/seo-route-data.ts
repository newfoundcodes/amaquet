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

import { defineRouteMiddleware } from "@astrojs/starlight/route-data"
import { getPageDescription, routePath } from "./utils/seo.mjs"

const SECTION_LABELS: Record<string, string> = {
  concepts: "Concepts",
  "data-types": "Data types",
  development: "Development",
  "getting-started": "Getting started",
  operations: "Operations",
  protocol: "Protocol",
  reference: "Reference",
}

function setMeta(
  head: App.Locals["starlightRoute"]["head"],
  attribute: "name" | "property",
  key: string,
  content: string,
) {
  const existing = head.find((entry) => entry.tag === "meta" && entry.attrs?.[attribute] === key)
  if (existing) {
    existing.attrs = { ...existing.attrs, [attribute]: key, content }
  } else {
    head.push({ tag: "meta", attrs: { [attribute]: key, content } })
  }
}

function absoluteUrl(path: string, siteUrl: string) {
  if (/^https?:\/\//i.test(path)) return path
  const base = /^https?:\/\//i.test(siteUrl) ? siteUrl.replace(/\/+$/, "") : "https://amaquet.newfoundcodes.com"
  return `${base}/${path.replace(/^\/+/, "")}`
}

function getSiteUrl(context: { site?: URL; url: URL }) {
  const candidates = [context.site?.href, context.url.origin, "https://amaquet.newfoundcodes.com/"]
  for (const candidate of candidates) {
    try {
      return new URL(candidate ?? "").href
    } catch {
      continue
    }
  }
  return "https://amaquet.newfoundcodes.com/"
}

function getBreadcrumbs(id: string, title: string, siteUrl: string) {
  if (id === "index") return []

  const segments = id.split("/")
  const visibleSegments = segments.filter((segment) => segment !== "index")
  const items = [{ name: "Amaquet documentation", item: absoluteUrl("/", siteUrl) }]
  let visibleIndex = 0

  segments.forEach((segment, segmentIndex) => {
    if (segment === "index") return

    if (segment === "types") return

    const isLast = visibleIndex === visibleSegments.length - 1
    const name = isLast
      ? title
      : (SECTION_LABELS[segment] ?? segment.replaceAll("_", " ").replace(/\b\w/g, (letter) => letter.toUpperCase()))
    const routeId = segments.slice(0, segmentIndex + 1).join("/")
    items.push({ name, item: absoluteUrl(routePath(routeId), siteUrl) })
    visibleIndex += 1
  })

  return items
}

export const onRequest = defineRouteMiddleware((context) => {
  const route = context.locals.starlightRoute
  const { entry } = route
  const seoId = entry.id || "index"
  const description = getPageDescription({
    id: seoId,
    title: entry.data.title,
    body: entry.body,
    description: entry.data.description,
  })

  entry.data.description = description
  setMeta(route.head, "name", "description", description)
  setMeta(route.head, "property", "og:description", description)
  setMeta(route.head, "name", "twitter:description", description)
  setMeta(route.head, "name", "twitter:title", entry.data.title)
  setMeta(route.head, "name", "twitter:card", "summary_large_image")

  const siteUrl = getSiteUrl(context)
  const socialImage = absoluteUrl("/og-image.png", siteUrl)
  setMeta(route.head, "property", "og:image", socialImage)
  setMeta(route.head, "property", "og:image:alt", "Amaquet documentation")
  setMeta(route.head, "property", "og:image:type", "image/png")
  setMeta(route.head, "property", "og:image:width", "1200")
  setMeta(route.head, "property", "og:image:height", "630")
  setMeta(route.head, "name", "twitter:image", socialImage)
  setMeta(route.head, "name", "twitter:image:alt", "Amaquet documentation")

  const isNotFound = seoId === "404"
  setMeta(
    route.head,
    "name",
    "robots",
    isNotFound ? "noindex, nofollow" : "index, follow, max-image-preview:large, max-snippet:-1, max-video-preview:-1",
  )

  if (isNotFound) return

  const canonical = route.head.find((item) => item.tag === "link" && item.attrs?.rel === "canonical")?.attrs?.href
  const pageUrl = absoluteUrl(routePath(seoId), siteUrl)
  const canonicalUrl = canonical ? absoluteUrl(canonical, siteUrl) : pageUrl
  const structuredData =
    seoId === "index"
      ? {
          "@context": "https://schema.org",
          "@type": "WebSite",
          name: "Amaquet Documentation",
          description,
          url: canonicalUrl,
          inLanguage: route.lang,
        }
      : {
          "@context": "https://schema.org",
          "@graph": [
            {
              "@type": "TechArticle",
              headline: entry.data.title,
              description,
              url: canonicalUrl,
              mainEntityOfPage: canonicalUrl,
              inLanguage: route.lang,
              isPartOf: {
                "@type": "WebSite",
                name: "Amaquet Documentation",
                url: siteUrl,
              },
              breadcrumb: { "@id": `${canonicalUrl}#breadcrumb` },
            },
            {
              "@type": "BreadcrumbList",
              "@id": `${canonicalUrl}#breadcrumb`,
              itemListElement: getBreadcrumbs(seoId, entry.data.title, siteUrl).map((item, index) => ({
                "@type": "ListItem",
                position: index + 1,
                ...item,
              })),
            },
          ],
        }

  route.head.push({
    tag: "script",
    attrs: { type: "application/ld+json" },
    content: JSON.stringify(structuredData).replaceAll("<", "\\u003c"),
  })
})
