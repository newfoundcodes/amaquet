import { readdirSync, readFileSync, statSync } from "node:fs"
import path from "node:path"

const docsRoot = path.resolve(".")
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
const errors = []

function walk(directory) {
  return readdirSync(directory).flatMap((name) => {
    const entry = path.join(directory, name)
    return statSync(entry).isDirectory() ? walk(entry) : [entry]
  })
}

function isStructuralBlock(line) {
  return /^(#{1,6})\s|^```|^~~~|^\||^[-*+] |^\d+\. |^:::|^<|^---$/.test(line)
}

function checkFile(file) {
  const lines = readFileSync(file, "utf8").split("\n")
  let fence = ""
  let bodyStart = 0

  if (lines[0] === "---") {
    bodyStart = 1
    while (bodyStart < lines.length && lines[bodyStart] !== "---") bodyStart++
    bodyStart++
    while (bodyStart < lines.length && !lines[bodyStart].trim()) bodyStart++

    if (bodyStart === lines.length || isStructuralBlock(lines[bodyStart])) {
      errors.push(`${path.relative(docsRoot, file)}: page title needs an introductory paragraph`)
    }
  }

  for (let index = bodyStart; index < lines.length; index++) {
    const line = lines[index]
    const marker = line.match(/^\s*(```|~~~)/)?.[1]
    if (marker) {
      fence = fence === marker ? "" : fence || marker
      continue
    }
    if (fence || !/^(#{1,6})\s+/.test(line)) continue

    let next = index + 1
    while (next < lines.length && !lines[next].trim()) next++
    if (next === lines.length || isStructuralBlock(lines[next])) {
      errors.push(`${path.relative(docsRoot, file)}:${index + 1}: heading needs an introductory paragraph`)
    }
  }
}

const sourceFiles = [
  ...topLevelPages.map((name) => path.join(docsRoot, name)),
  ...contentDirectories.flatMap((directory) =>
    walk(path.join(docsRoot, directory)).filter((file) => file.endsWith(".md")),
  ),
]

for (const file of sourceFiles) checkFile(file)

if (errors.length) {
  console.error(errors.map((error) => `ERROR: ${error}`).join("\n"))
  process.exit(1)
}

console.log("documentation heading-description check passed")
