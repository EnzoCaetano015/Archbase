import assert from "node:assert/strict"
import { readFile, stat } from "node:fs/promises"
import { createHash } from "node:crypto"
import {
  pages,
  navGroups,
  pageMap,
  docAliases,
  catalogFamilies,
  catalogRules,
  skills,
  sourceUrl,
} from "../src/content.ts"

const site = new URL("../", import.meta.url)
const root = new URL("../../", import.meta.url)
const load = (file) => readFile(new URL(file, root))
assert.equal(new Set(pages.map((page) => page.slug)).size, pages.length)
for (const group of navGroups)
  for (const page of group.pages)
    assert(pageMap.has(page.slug), `Broken navigation: ${page.slug}`)
for (const page of pages) {
  assert.equal(
    new Set(page.sections.map((section) => section.id)).size,
    page.sections.length,
    `Duplicate anchors: ${page.slug}`,
  )
  const source = sourceUrl(page).split("/blob/main/")[1]
  await stat(new URL(source, root))
  const html = await readFile(
    new URL(`dist/docs/${page.slug}/index.html`, site),
    "utf8",
  )
  assert(
    html.includes(
      `<link rel="canonical" href="https://archbase.caetanodev.com/docs/${page.slug}"`,
    ),
  )
  assert(
    html.includes("<main>") && html.includes(page.title.replace(/&/g, "&amp;")),
    `Missing static content: ${page.slug}`,
  )
}
for (const [alias, target] of Object.entries(docAliases)) {
  const [slug, anchor] = target.split("#")
  const page = pageMap.get(slug)
  assert(page, `Broken alias ${alias}`)
  if (anchor)
    assert(
      page.sections.some((section) => section.id === anchor),
      `Broken alias anchor ${alias}: ${target}`,
    )
}
assert.equal(catalogRules.length, 14)
for (const family of catalogFamilies)
  for (const id of family.rules)
    assert(catalogRules.some((rule) => rule.id === id))
let count = 0
for (const rule of catalogRules) {
  await stat(new URL(rule.source, root))
  await stat(new URL(rule.notesUrl.split("/blob/main/")[1], root))
  assert(rule.examples.length, `No examples for ${rule.id}`)
  for (const example of rule.examples) {
    const raw = await load(example.source)
    assert.equal(
      createHash("sha256").update(raw).digest("hex"),
      example.sha256,
      `Example drift: ${example.source}`,
    )
    assert(
      raw.toString("utf8").replace(/\r\n/g, "\n").includes(example.code),
      `Example differs: ${example.source}`,
    )
    count++
  }
}
assert.equal(skills.length, 11)
assert.equal(new Set(skills.map((skill) => skill.url)).size, 11)
for (const skill of skills) {
  assert.equal(skill.status, "Published")
  assert.match(
    skill.url,
    /^https:\/\/(www\.)?agensi\.io\/skills\/archbase-[a-z-]+$/,
  )
  assert(pageMap.has(skill.slug))
}
for (const file of ["install.sh", "install.ps1"]) {
  const original = await readFile(new URL(file, site))
  assert(
    original.equals(await readFile(new URL(`public/${file}`, site))),
    `Installer source drift: ${file}`,
  )
  assert(
    original.equals(await readFile(new URL(`dist/${file}`, site))),
    `Installer output drift: ${file}`,
  )
}
await stat(new URL("dist/assets/archbase-mark.svg", site))
console.log(
  `Checked ${pages.length} pages, ${Object.keys(docAliases).length} aliases, ${count} exact examples, 11 listings and both unchanged installers.`,
)
