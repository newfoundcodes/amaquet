import { existsSync, readdirSync, readFileSync, statSync } from "node:fs"
import path from "node:path"

const root = path.resolve("dist")
const base = "/"
const origin = "https://amaquet.invalid"
const errors = []
const descriptions = new Map()

function walk(directory) {
  return readdirSync(directory).flatMap((name) => {
    const entry = path.join(directory, name)
    return statSync(entry).isDirectory() ? walk(entry) : [entry]
  })
}

function localTarget(pathname) {
  const relative = decodeURIComponent(pathname.slice(base.length))
  if (relative === "") return path.join(root, "index.html")
  if (relative.endsWith("/")) return path.join(root, relative, "index.html")
  return path.join(root, relative)
}

function metaContent(html, attribute, key) {
  for (const match of html.matchAll(/<meta\b[^>]*>/g)) {
    const tag = match[0]
    if (!new RegExp(`\\b${attribute}=["']${key}["']`).test(tag)) continue
    return tag.match(/\bcontent=(["'])(.*?)\1/)?.[2]
  }
}

for (const required of ["llms.txt", "llms-full.txt", "robots.txt", "sitemap.xml"]) {
  if (!existsSync(path.join(root, required))) errors.push(`missing generated discovery file /${required}`)
}

for (const page of walk(root).filter((file) => file.endsWith(".html"))) {
  const pagePath = path.relative(root, page).split(path.sep).join("/")
  const pageUrl = new URL(pagePath === "index.html" ? base : `${base}${pagePath.replace(/index\.html$/, "")}`, origin)
  const html = readFileSync(page, "utf8")

  if (!/<html[^>]*data-theme=["']dark["']/.test(html)) {
    errors.push(`${pagePath}: documentation page is not locked to dark mode`)
  }
  if (html.includes("<starlight-theme-select")) {
    errors.push(`${pagePath}: unexpected light/dark theme selector`)
  }
  if (!/<meta[^>]*name=["']color-scheme["'][^>]*content=["']dark["']/.test(html)) {
    errors.push(`${pagePath}: missing dark-only color-scheme metadata`)
  }

  const description = metaContent(html, "name", "description")
  if (!description) {
    errors.push(`${pagePath}: missing page description`)
  } else if (pagePath !== "404.html") {
    if (description.length < 50 || description.length > 160) {
      errors.push(`${pagePath}: description length is ${description.length}; expected 50–160 characters`)
    }
    const duplicate = descriptions.get(description)
    if (duplicate) errors.push(`${pagePath}: duplicates the description from ${duplicate}`)
    descriptions.set(description, pagePath)
  }

  const robots = metaContent(html, "name", "robots")
  if (!robots || (pagePath === "404.html" ? !robots.includes("noindex") : !robots.includes("index"))) {
    errors.push(`${pagePath}: missing expected robots directive`)
  }

  const socialImage = metaContent(html, "property", "og:image")
  if (!socialImage || !socialImage.startsWith("https://")) {
    errors.push(`${pagePath}: missing absolute Open Graph image`)
  }

  const twitterCard = metaContent(html, "name", "twitter:card")
  if (twitterCard !== "summary_large_image") {
    errors.push(`${pagePath}: missing summary_large_image Twitter card`)
  }

  if (pagePath !== "404.html") {
    const structuredData = html.match(/<script[^>]*type=["']application\/ld\+json["'][^>]*>([\s\S]*?)<\/script>/)?.[1]
    if (!structuredData) {
      errors.push(`${pagePath}: missing JSON-LD structured data`)
    } else {
      try {
        const parsed = JSON.parse(structuredData)
        if (pagePath !== "index.html") {
          const graph = Array.isArray(parsed["@graph"]) ? parsed["@graph"] : []
          const breadcrumbs = graph.find((item) => item["@type"] === "BreadcrumbList")
          if (!breadcrumbs?.itemListElement?.length) {
            errors.push(`${pagePath}: missing breadcrumb structured data`)
          }
        }
      } catch {
        errors.push(`${pagePath}: invalid JSON-LD structured data`)
      }
    }
  }

  for (const [, href] of html.matchAll(/\b(?:href|src)=["']([^"']+)["']/g)) {
    if (!href || href.startsWith("#") || href.startsWith("mailto:") || href.startsWith("tel:")) continue
    const url = new URL(href, pageUrl)
    if (url.origin !== origin || !url.pathname.startsWith(base)) continue

    const target = localTarget(url.pathname)
    if (!existsSync(target)) errors.push(`${pagePath}: broken link ${href}`)
  }
}

const llmsPath = path.join(root, "llms.txt")
if (existsSync(llmsPath)) {
  const llms = readFileSync(llmsPath, "utf8")
  if (!llms.startsWith("# Amaquet Documentation\n\n> ")) errors.push("llms.txt: invalid llms.txt heading or summary")

  const links = [...llms.matchAll(/^- \[[^\]]+\]\(([^)]+)\): /gm)].map((match) => match[1])
  if (links.length !== 145) {
    errors.push(`llms.txt: expected 145 resource links, found ${links.length}`)
  }

  for (const href of links) {
    const url = new URL(href, origin)
    if (url.origin === origin && !existsSync(localTarget(url.pathname))) errors.push(`llms.txt: broken link ${href}`)
  }
}

const fullPath = path.join(root, "llms-full.txt")
if (existsSync(fullPath)) {
  const full = readFileSync(fullPath, "utf8")
  const documentCount = [...full.matchAll(/^Source: .*$/gm)].length

  if (documentCount !== 144) {
    errors.push(`llms-full.txt: expected 144 documents, found ${documentCount}`)
  }
}

const sitemapPath = path.join(root, "sitemap.xml")
if (existsSync(sitemapPath)) {
  const sitemap = readFileSync(sitemapPath, "utf8")
  const locations = [...sitemap.matchAll(/<loc>([^<]+)<\/loc>/g)].map((match) => match[1])

  if (locations.length !== 144) {
    errors.push(`sitemap.xml: expected 144 URLs, found ${locations.length}`)
  }

  for (const location of locations) {
    const url = new URL(location)
    if (!existsSync(localTarget(url.pathname))) errors.push(`sitemap.xml: broken URL ${location}`)
  }
}

const robotsPath = path.join(root, "robots.txt")
if (existsSync(robotsPath) && !readFileSync(robotsPath, "utf8").includes("/sitemap.xml")) {
  errors.push("robots.txt: missing sitemap.xml discovery URL")
}

if (errors.length) {
  console.error(errors.map((error) => `ERROR: ${error}`).join("\n"))
  process.exit(1)
}

console.log("built documentation link check passed")
