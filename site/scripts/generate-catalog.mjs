import { readFile, writeFile, mkdir } from "node:fs/promises"
import { fileURLToPath } from "node:url"
import path from "node:path"
import { createHash } from "node:crypto"
import { parse } from "yaml"

const root = fileURLToPath(new URL("../../", import.meta.url))
const read = async (file) => readFile(path.join(root, file), "utf8")
const github = "https://github.com/EnzoCaetano015/Archbase/blob/main/"
const index = parse(await read("registry/index.yaml"))
const rules = parse(await read("registry/rules/index.yaml"))
const patterns = new Map(index.patterns.map((entry) => [entry.id, entry]))
const documents = []
for (const entry of rules.rules) {
  const rulePath = `registry/rules/${entry.path}/rule.yaml`
  const rule = parse(await read(rulePath))
  if (rule.id !== entry.id) throw new Error(`RuleID mismatch: ${rulePath}`)
  const examples = []
  for (const id of [...new Set(rule.scopes.map((scope) => scope.pattern))]) {
    const item = patterns.get(id)
    if (!item) throw new Error(`Unknown PatternID ${id} in ${rulePath}`)
    const manifestPath = `registry/${item.path}/manifest.yaml`
    const manifest = parse(await read(manifestPath))
    for (const file of manifest.structure.files) {
      const source = `registry/${item.path}/${file.source}`
      let text
      try {
        text = await read(source)
      } catch (error) {
        if (!file.required && error.code === "ENOENT") continue
        throw error
      }
      const fence = text.trim().match(/^(`{3,})([^\n]*)\n([\s\S]*)\n\1$/)
      if (file.format === "markdown-code" && !fence)
        throw new Error(`Invalid typed example: ${source}`)
      examples.push({
        patternId: id,
        name: manifest.name,
        source,
        url: github + source,
        manifestUrl: github + manifestPath,
        destination: file.destination,
        language: fence ? fence[2].trim() : "text",
        code: fence ? fence[3].replace(/\r\n/g, "\n") : text,
        sha256: createHash("sha256").update(text).digest("hex"),
      })
    }
  }
  documents.push({
    id: rule.id,
    name: rule.name,
    description: rule.description,
    version: rule.version,
    source: rulePath,
    url: github + rulePath,
    notesUrl: github + `registry/rules/${entry.path}/README.md`,
    scopes: rule.scopes,
    restrictions: rule.rules,
    examples,
  })
}
await mkdir(new URL("../src/generated/", import.meta.url), { recursive: true })
await writeFile(
  new URL("../src/generated/catalog.json", import.meta.url),
  JSON.stringify(documents, null, 2) + "\n",
)
console.log(
  `Generated ${documents.length} rules and ${documents.reduce((n, r) => n + r.examples.length, 0)} canonical examples.`,
)
