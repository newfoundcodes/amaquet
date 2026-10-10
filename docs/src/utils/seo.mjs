const MAX_DESCRIPTION_LENGTH = 160
const PAGE_DESCRIPTIONS = {
  index:
    "Official Amaquet documentation for installing, configuring, operating, and integrating with the standalone typed in-memory database and its binary protocol.",
  architecture:
    "Amaquet architecture reference covering the sharded storage engine, typed values, protocol server, Go client, HTTP control plane, and append-only persistence.",
  protocol:
    "Amaquet protocol v1 reference for connection URIs, binary frames, multiplexing, negotiation, authentication, commands, Pub/Sub events, and RBAC.",
}

function plainText(markdown) {
  return markdown
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/<[^>]+>/g, " ")
    .replace(/[`*_~]/g, "")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/\s+/g, " ")
    .trim()
}

function truncateDescription(description) {
  if (description.length <= MAX_DESCRIPTION_LENGTH) return description

  const sentences = description.match(/.*?[.!?](?:\s|$)/g) ?? []
  const firstSentence = sentences[0]?.trim()
  if (firstSentence && firstSentence.length >= 70 && firstSentence.length <= MAX_DESCRIPTION_LENGTH) {
    return firstSentence
  }

  const shortened = description.slice(0, MAX_DESCRIPTION_LENGTH - 1)
  const lastSpace = shortened.lastIndexOf(" ")
  return `${shortened.slice(0, lastSpace > 100 ? lastSpace : shortened.length).trimEnd()}…`
}

function firstProseParagraph(markdown) {
  const withoutCode = markdown.replace(/```[\s\S]*?```/g, "").replace(/~~~[\s\S]*?~~~/g, "")

  for (const block of withoutCode.split(/\n\s*\n/)) {
    const candidate = block.trim()
    if (!candidate) continue
    if (/^(?:#{1,6}\s|[-*+]\s|\d+\.\s|\||>|::|<|\{)/.test(candidate)) continue
    if (/^\*\*[^*]+:\*\*/.test(candidate)) continue

    const text = plainText(candidate)
    if (text.length >= 40 && !text.endsWith(":")) return text
  }

  return ""
}

function readableTypeName(title) {
  return title.replaceAll("_", " ")
}

export function getPageDescription({ id, title, body, description }) {
  const pageId = id || "index"
  if (PAGE_DESCRIPTIONS[pageId]) return PAGE_DESCRIPTIONS[pageId]
  if (description) return truncateDescription(plainText(description))

  if (pageId.startsWith("data-types/types/")) {
    return truncateDescription(
      `Amaquet ${readableTypeName(title)} data type reference covering construction, supported operations, key lifecycle, contract tests, and fuzzing.`,
    )
  }

  const paragraph = firstProseParagraph(body)
  if (paragraph) return truncateDescription(paragraph)

  return truncateDescription(
    `Learn about ${title} in the official Amaquet documentation, including implementation-aligned behavior, configuration, examples, and operational guidance.`,
  )
}

export function routePath(id) {
  return id === "index" ? "/" : `/${id.replace(/\/index$/, "")}/`
}

export function parseMarkdownSource(source) {
  const match = source.match(/^---\n([\s\S]*?)\n---\n?([\s\S]*)$/)
  if (!match) return { body: source, data: {} }

  const data = {}
  for (const line of match[1].split("\n")) {
    const field = line.match(/^([A-Za-z][A-Za-z0-9_-]*):\s*(.*)$/)
    if (!field) continue

    let value = field[2].trim()
    if (value.startsWith('"') && value.endsWith('"')) {
      try {
        value = JSON.parse(value)
      } catch {
        value = value.slice(1, -1)
      }
    } else if (value.startsWith("'") && value.endsWith("'")) {
      value = value.slice(1, -1)
    } else if (value === "true" || value === "false") {
      value = value === "true"
    }
    data[field[1]] = value
  }

  return { body: match[2].trim(), data }
}
