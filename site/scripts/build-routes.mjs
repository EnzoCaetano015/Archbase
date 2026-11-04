import { readFile, writeFile, mkdir } from "node:fs/promises"
import { pages, skills, docAliases, catalogRules } from "../src/content.ts"

const output = new URL("../dist/", import.meta.url)
const template = await readFile(new URL("index.html", output), "utf8")
const origin = "https://archbase.caetanodev.com"
const escape = (text) =>
  String(text).replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ],
  )
const sectionHTML = (section) =>
  `<section id="${escape(section.id)}"><h2>${escape(section.title)}</h2><p>${escape(section.body)}</p>${
    section.code ? `<pre><code>${escape(section.code)}</code></pre>` : ""
  }${(section.links ?? []).map((link) => `<a href="${escape(link.href)}">${escape(link.label)}</a>`).join(" ")}</section>`
async function emit(route, title, description, content, canonical = route) {
  const directory = new URL(`${route.replace(/^\//, "")}/`, output)
  await mkdir(directory, { recursive: true })
  const html = template
    .replace(
      /<title>.*?<\/title>/,
      `<title>${escape(title)} · Archbase</title>`,
    )
    .replace(
      /<meta name="description" content="[^"]*"\s*\/>/,
      `<meta name="description" content="${escape(description)}" />`,
    )
    .replace(
      /<link rel="canonical" href="[^"]*"\s*\/>/,
      `<link rel="canonical" href="${origin}${canonical}" />`,
    )
    .replace(
      '<div id="root"></div>',
      `<div id="root"><main><h1>${escape(title)}</h1><p>${escape(description)}</p>${content}</main></div>`,
    )
  await writeFile(new URL("index.html", directory), html)
}
for (const page of pages) {
  let content = page.sections.map(sectionHTML).join("")
  if (page.slug.startsWith("catalog-")) {
    const family = page.slug.slice(8)
    const related = catalogRules.filter((rule) =>
      (
        ({
          nextjs: ["next-"],
          react: ["react-"],
          nestjs: ["nestjs-"],
          flutter: ["flutter-"],
          "spring-boot": ["spring-boot-"],
          dotnet: ["dotnet-"],
          fastapi: ["python-"],
          astro: ["astro-"],
          discordjs: ["discordjs-"],
        })[family] ?? []
      ).some((prefix) => rule.id.startsWith(`architecture/${prefix}`)),
    )
    content += related
      .map(
        (rule) =>
          `<section><h2>${escape(rule.name)}</h2><a href="${rule.url}">Canonical rule</a>${rule.examples.map((example) => `<details><summary>${escape(example.patternId)} — ${escape(example.source)}</summary><a href="${example.url}">Original example</a><pre><code>${escape(example.code)}</code></pre></details>`).join("")}</section>`,
      )
      .join("")
  }
  await emit(`/docs/${page.slug}`, page.title, page.description, content)
}
await emit(
  "/skills",
  "Archbase Skills",
  "Free orientation and Premium architecture workflows. All eleven skills are published on Agensi.",
  skills
    .map(
      (skill) =>
        `<section><h2>${escape(skill.name)}</h2><p>${escape(skill.summary)}</p><p>${escape(skill.price)} · ${escape(skill.status)}</p><a href="${skill.url}">View on Agensi</a></section>`,
    )
    .join(""),
)
for (const [alias, target] of Object.entries(docAliases)) {
  const page = pages.find((page) => page.slug === target.split("#")[0])
  if (!page) throw new Error(`Broken alias ${alias}`)
  await emit(
    `/docs/${alias}`,
    page.title,
    page.description,
    page.sections.map(sectionHTML).join(""),
    `/docs/${target.split("#")[0]}`,
  )
}
const routes = ["/", "/skills/", ...pages.map((page) => `/docs/${page.slug}`)]
await writeFile(
  new URL("sitemap.xml", output),
  `<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${routes.map((route) => `<url><loc>${origin}${route}</loc></url>`).join("")}</urlset>\n`,
)
console.log(
  `Built ${routes.length} static routes plus ${Object.keys(docAliases).length} compatibility routes.`,
)
