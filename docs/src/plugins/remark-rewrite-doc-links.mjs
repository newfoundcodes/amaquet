import path from "node:path"

const markdownLink = /^([^?#]+\.md)([?#].*)?$/

function routeFor(source) {
  const normalized = source.split(path.sep).join("/")
  if (normalized === "index.md") return ""
  if (normalized.endsWith("/index.md")) return normalized.slice(0, -"/index.md".length)
  return normalized.slice(0, -".md".length)
}

function visit(node, callback) {
  callback(node)
  if (Array.isArray(node.children)) {
    for (const child of node.children) visit(child, callback)
  }
}

export default function rewriteDocLinks() {
  return (tree, file) => {
    const projectRoot = process.cwd()
    const source = path.relative(projectRoot, file.path)
    const sourceRoute = routeFor(source)

    visit(tree, (node) => {
      if ((node.type !== "link" && node.type !== "definition") || typeof node.url !== "string") return
      const match = node.url.match(markdownLink)
      if (!match) return

      const target = path.resolve(path.dirname(file.path), match[1])
      const relativeTarget = path.relative(projectRoot, target)
      if (relativeTarget.startsWith("..") || path.isAbsolute(relativeTarget)) return

      const targetRoute = routeFor(relativeTarget)
      const relativeRoute = path.posix.relative(sourceRoute || ".", targetRoute || ".")
      node.url = `${relativeRoute || "."}/${match[2] || ""}`
    })
  }
}
