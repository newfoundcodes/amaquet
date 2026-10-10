import { existsSync, mkdirSync, readFileSync, readdirSync, statSync, writeFileSync } from "node:fs"
import { execFileSync } from "node:child_process"
import path from "node:path"
import { fileURLToPath } from "node:url"
import { getPageDescription, parseMarkdownSource, routePath } from "../src/utils/seo.mjs"

const docsRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..")
const publicRoot = path.join(docsRoot, "public")
const checkOnly = process.argv.includes("--check")
const site = new URL(process.env.DOCS_SITE_URL || "https://amaquet.newfoundcodes.com")
const contentDirectories = [
  "concepts",
  "data-types",
  "development",
  "getting-started",
  "operations",
  "protocol",
  "reference",
]
const topLevelPages = ["index.md", "architecture.md", "protocol.md"]

function walk(directory) {
  return readdirSync(directory).flatMap((name) => {
    const entry = path.join(directory, name)
    return statSync(entry).isDirectory() ? walk(entry) : [entry]
  })
}

function pageId(file) {
  return path.relative(docsRoot, file).split(path.sep).join("/").replace(/\.md$/, "")
}

function shiftHeadings(markdown) {
  let fence = ""
  return markdown
    .split("\n")
    .map((line) => {
      const marker = line.match(/^\s*(```|~~~)/)?.[1]
      if (marker) {
        fence = fence === marker ? "" : fence || marker
        return line
      }
      if (fence) return line
      return line.replace(/^(#{1,5})(\s+)/, "#$1$2")
    })
    .join("\n")
}

function rewriteLocalLinks(markdown, id) {
  const directory = path.posix.dirname(id)
  return markdown.replace(/\]\(([^)\s]+\.md)(#[^)]+)?\)/g, (match, target, hash = "") => {
    if (/^(?:[a-z]+:|\/)/i.test(target)) return match
    const resolved = path.posix.normalize(path.posix.join(directory, target)).replace(/\.md$/, "")
    return `](${routePath(resolved)}${hash})`
  })
}

function sectionFor(id) {
  if (id === "index" || id === "architecture" || id === "protocol") return "Core documentation"
  if (id.startsWith("getting-started/")) return "Getting started"
  if (id.startsWith("concepts/")) return "Concepts"
  if (id.startsWith("protocol/")) return "Protocol"
  if (id.startsWith("data-types/types/")) return "Data type reference"
  if (id.startsWith("data-types/")) return "Data types"
  if (id.startsWith("operations/")) return "Operations"
  if (id.startsWith("development/")) return "Development"
  return "Reference"
}

function lastModified(file) {
  try {
    const repositoryDate = execFileSync("git", ["log", "-1", "--format=%cI", "--", file], {
      cwd: docsRoot,
      encoding: "utf8",
    }).trim()
    if (!repositoryDate) return undefined

    try {
      execFileSync("git", ["diff", "--quiet", "HEAD", "--", file], { cwd: docsRoot, stdio: "ignore" })
    } catch {
      return new Date().toISOString().slice(0, 10)
    }

    return repositoryDate.slice(0, 10)
  } catch {
    return undefined
  }
}

const sourceFiles = [
  ...topLevelPages.map((name) => path.join(docsRoot, name)),
  ...contentDirectories.flatMap((directory) =>
    walk(path.join(docsRoot, directory)).filter((file) => file.endsWith(".md")),
  ),
]

const pages = sourceFiles
  .map((file) => {
    const id = pageId(file)
    const { body, data } = parseMarkdownSource(readFileSync(file, "utf8"))
    return {
      id,
      title: data.title,
      body,
      description: getPageDescription({ id, title: data.title, body, description: data.description }),
      draft: data.draft === true,
      lastmod: lastModified(file),
    }
  })
  .filter((page) => !page.draft && page.title)
  .sort((left, right) => routePath(left.id).localeCompare(routePath(right.id)))

const sections = new Map()
for (const page of pages) {
  const section = sectionFor(page.id)
  if (!sections.has(section)) sections.set(section, [])
  sections.get(section).push(page)
}

const sectionOrder = [
  "Core documentation",
  "Getting started",
  "Concepts",
  "Protocol",
  "Data types",
  "Data type reference",
  "Operations",
  "Development",
  "Reference",
]

const llms = [
  "# Amaquet Documentation",
  "",
  "> Official documentation for Amaquet, a standalone typed in-memory database written in Go with its own binary protocol, persistence layer, Go client, and HTTP administration API.",
  "",
  "Use these implementation-aligned pages for installation, protocol integration, data-type behavior, operations, and development. The complete documentation is also available as one machine-readable file at `/llms-full.txt`.",
  "",
  "## Complete export",
  "",
  "- [Amaquet documentation — full text](/llms-full.txt): Complete Markdown content for every published documentation page in a single file.",
]

for (const section of sectionOrder) {
  const entries = sections.get(section)
  if (!entries?.length) continue
  llms.push("", `## ${section}`, "")
  for (const page of entries) llms.push(`- [${page.title}](${routePath(page.id)}): ${page.description}`)
}

const full = [
  "# Amaquet Documentation — Full Text",
  "",
  "> Complete, build-generated Markdown export of the official Amaquet documentation for language models, retrieval systems, and offline reference.",
  "",
  "Each document includes its canonical site path and an implementation-aligned summary. Source order follows the public documentation route structure.",
]

for (const page of pages) {
  full.push(
    "",
    "---",
    "",
    `## ${page.title}`,
    "",
    `Source: [canonical documentation page](${routePath(page.id)})`,
    "",
    `Summary: ${page.description}`,
    "",
    shiftHeadings(rewriteLocalLinks(page.body, page.id)),
  )
}

const sitemap = [
  '<?xml version="1.0" encoding="UTF-8"?>',
  '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
  ...pages.map((page) => {
    const lastmod = page.lastmod ? `<lastmod>${page.lastmod}</lastmod>` : ""
    return `  <url><loc>${new URL(routePath(page.id), site).href}</loc>${lastmod}</url>`
  }),
  "</urlset>",
  "",
].join("\n")

const outputs = new Map([
  ["llms.txt", `${llms.join("\n")}\n`],
  ["llms-full.txt", `${full.join("\n")}\n`],
  ["sitemap.xml", sitemap],
  ["robots.txt", `User-agent: *\nAllow: /\n\nSitemap: ${new URL("/sitemap.xml", site).href}\n`],
])

mkdirSync(publicRoot, { recursive: true })
let stale = false
for (const [name, content] of outputs) {
  const target = path.join(publicRoot, name)
  if (checkOnly) {
    if (!existsSync(target) || readFileSync(target, "utf8") !== content) {
      console.error(`${path.relative(docsRoot, target)} is missing or stale; run npm run generate:ai`)
      stale = true
    }
  } else {
    writeFileSync(target, content)
  }
}

if (stale) process.exit(1)
if (!checkOnly) console.log(`generated AI indexes and sitemap.xml from ${pages.length} documentation pages`)
