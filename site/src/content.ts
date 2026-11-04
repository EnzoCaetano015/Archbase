import catalogDocuments from "./generated/catalog.json" with { type: "json" };
export type Section = {
    id: string;
    title: string;
    body: string;
    code?: string;
    language?: string;
    links?: Array<{
        label: string;
        href: string;
    }>;
    callout?: string;
};
export type DocPage = {
    slug: string;
    title: string;
    description: string;
    group: string;
    sections: Section[];
    source?: string;
};
export type NavGroup = {
    title: string;
    pages: Array<{
        title: string;
        slug: string;
    }>;
};
const sourceRoot = "https://github.com/EnzoCaetano015/Archbase/blob/main/";
const navSpec: Array<[
    string,
    Array<[
        string,
        string
    ]>
]> = [
    [
        "Start here",
        [
            ["Introduction", "introduction"],
            ["Quick start", "quick-start"],
            ["Installation & updates", "installation"],
        ],
    ],
    [
        "Guides",
        [
            ["Patterns, rules & scopes", "concepts"],
            ["CLI & registries", "cli"],
            ["Agent instructions", "agents"],
            ["Read-only MCP", "mcp-setup"],
        ],
    ],
    [
        "Catalog",
        [
            ["Overview", "catalog"],
            ["Next.js", "catalog-nextjs"],
            ["React", "catalog-react"],
            ["NestJS", "catalog-nestjs"],
            ["Flutter", "catalog-flutter"],
            ["Spring Boot", "catalog-spring-boot"],
            [".NET", "catalog-dotnet"],
            ["FastAPI", "catalog-fastapi"],
            ["Astro", "catalog-astro"],
            ["Discord.js", "catalog-discordjs"],
        ],
    ],
    ["Skills", [["Skills & availability", "skills"]]],
    [
        "Reference",
        [
            ["Schemas", "schemas"],
            ["Project & privacy", "project"],
        ],
    ],
];
export const navGroups: NavGroup[] = navSpec.map(([title, pages]) => ({
    title,
    pages: pages.map(([pageTitle, slug]) => ({ title: pageTitle, slug })),
}));
const basePages: DocPage[] = [
    {
        slug: "introduction",
        title: "Introduction",
        description: "Make architecture conventions explicit for developers and AI coding agents.",
        group: "Getting started",
        source: "README.md",
        sections: [
            {
                id: "what-is-archbase",
                title: "What is Archbase?",
                body: "Archbase is an open-source Go CLI for sharing reusable code patterns and architecture rules. It gives humans and AI agents one explicit source for the conventions that already shape a project.",
            },
            {
                id: "four-parts",
                title: "Four complementary parts",
                body: "The CLI installs, queries, and manages patterns and rules. The public catalog provides references for supported stacks. The read-only MCP exposes resolved architecture context. Skills add guided workflows for orientation, auditing, planning, and review.",
            },
            {
                id: "what-it-is-not",
                title: "What Archbase does not replace",
                body: "Archbase does not replace your framework or impose a complete application. Patterns describe how a type of code is structured; rules describe responsibilities, paths, layer relationships, dependency constraints, and pattern associations.",
                callout: "Catalog conventions are starting points. Adopt them deliberately and customize local copies when the project requires it.",
            },
        ],
    },
    {
        slug: "installation",
        title: "Installation",
        description: "Install the official Linux build or preview builds for macOS and Windows.",
        group: "Getting started",
        source: "docs/installation.md",
        sections: [
            {
                id: "linux",
                title: "Linux — official channel",
                body: "Run the official installer. It selects the compatible release and verifies the downloaded artifact before installation.",
                code: "curl -fsSL https://archbase.caetanodev.com/install.sh | sh",
                language: "bash",
            },
            {
                id: "other-platforms",
                title: "macOS and Windows previews",
                body: "macOS and Windows binaries are preview builds without signing or certification. Review the release notes and checksums before installing them manually.",
                callout: "Do not bypass platform warnings without first verifying the checksum against the public release.",
            },
            {
                id: "verify",
                title: "Verify and update",
                body: "Confirm the executable and check for a newer stable release without changing the installation.",
                code: "arc version\narc update --check",
                language: "bash",
            },
            {
                id: "remove",
                title: "Manual installation and removal",
                body: "Release assets and checksums are published with each release. Place the verified executable on your PATH for a manual installation. Removing that executable uninstalls the CLI; Archbase does not modify project files during an update.",
            },
        ],
    },
    {
        slug: "quick-start",
        title: "Quick start",
        description: "Go from installation to resolved architecture context in a few commands.",
        group: "Getting started",
        source: "docs/getting-started.md",
        sections: [
            {
                id: "open",
                title: "1. Open Archbase",
                body: "Running arc without a subcommand opens the interactive terminal interface when the terminal is compatible.",
                code: "arc\narc version",
                language: "bash",
            },
            {
                id: "add",
                title: "2. Add a pattern",
                body: "Install a catalog pattern into the current project. arc add and arc create use the invocation directory as project root unless --project-root is provided.",
                code: "arc add next/page@1234 .",
                language: "bash",
            },
            {
                id: "resolve",
                title: "3. Resolve for a path",
                body: "Resolution reads .archbase/scopes.yaml and chooses the most specific matching directory scope.",
                code: "arc resolve src/pages/Home.tsx\narc inspect next/page@1234",
                language: "bash",
            },
            {
                id: "rules",
                title: "4. Adopt architecture rules",
                body: "Inspect a rule before adding it. Agent exports keep canonical rules under .archbase/rules/ and maintain a native index for the selected agent.",
                code: "arc rules list\narc rules inspect architecture/next-modular@1\narc rules add architecture/next-modular@1 --format agents",
                language: "bash",
            },
            {
                id: "next",
                title: "5. Connect an agent",
                body: "Export a native rule index, start the optional read-only MCP server, or use an Archbase skill for a guided workflow. These paths complement each other.",
            },
        ],
    },
    {
        slug: "terminal-ui",
        title: "Terminal UI",
        description: "Navigate the interactive terminal or switch to plain text output.",
        group: "Getting started",
        source: "docs/terminal-ui.md",
        sections: [
            {
                id: "launch",
                title: "Interactive mode",
                body: "Run arc in a compatible terminal to open the interface. Ctrl+P opens the command palette and Esc returns to the previous view.",
                code: "arc",
                language: "bash",
            },
            {
                id: "plain",
                title: "Plain output",
                body: "Use --plain when an interactive terminal is not desirable. Redirected output remains plain text so commands work in scripts and pipelines.",
                code: "arc --plain\narc rules list > rules.txt",
                language: "bash",
            },
            {
                id: "boundaries",
                title: "Command availability",
                body: "The terminal exposes implemented CLI operations. arc audit and arc doctor are proposals and are not available commands.",
                callout: "For exact flags, use the command help shipped with your installed release.",
            },
        ],
    },
    {
        slug: "patterns",
        title: "Patterns",
        description: "Patterns define how one type of code is structured.",
        group: "Core concepts",
        source: "docs/concepts.md",
        sections: [
            {
                id: "definition",
                title: "A focused code reference",
                body: "A pattern can describe pages, components, controllers, services, repositories, schemas, hooks, or another recurring type of code. It is not a complete application, feature generator, or product boilerplate.",
            },
            {
                id: "identity",
                title: "Stable identifiers",
                body: "PatternIDs use stack/type@id. The suffix identifies the entry; its semantic version is a separate manifest field.",
                code: "next/page@1234",
                language: "text",
            },
            {
                id: "inspect",
                title: "Inspect before adopting",
                body: "Inspect the manifest and declared files before adding a pattern to a scope.",
                code: "arc inspect next/page@1234\narc add next/page@1234 .",
                language: "bash",
            },
        ],
    },
    {
        slug: "architecture-rules",
        title: "Architecture rules",
        description: "Rules define responsibilities, paths, relationships, constraints, and pattern associations.",
        group: "Core concepts",
        source: "docs/rules.md",
        sections: [
            {
                id: "definition",
                title: "Project-level conventions",
                body: "Rules explain where code belongs, what each layer owns, which dependencies are allowed, and which patterns apply to paths. They reference patterns by identifier rather than duplicating every code example.",
            },
            {
                id: "identity",
                title: "Rule identifiers",
                body: "Architecture rules are versioned independently.",
                code: "architecture/next-feature-app-router@1",
                language: "text",
            },
            {
                id: "storage",
                title: "Canonical storage",
                body: "Added rules are stored once in .archbase/rules/. Agent-specific files receive indexes that point back to this canonical source.",
            },
        ],
    },
    {
        slug: "scopes-and-resolution",
        title: "Scopes and resolution",
        description: "Resolve the active pattern from the most specific matching directory scope.",
        group: "Core concepts",
        source: "docs/workspace.md",
        sections: [
            {
                id: "configuration",
                title: "Central configuration",
                body: "All scope declarations live in the project workspace.",
                code: ".archbase/scopes.yaml",
                language: "text",
            },
            {
                id: "precedence",
                title: "Most specific scope wins",
                body: "A scope for src/pages/admin can replace the pattern that applies in src/pages. Resolution is deterministic and based on directory specificity.",
            },
            {
                id: "errors",
                title: "Invalid scopes fail visibly",
                body: "An invalid matching scope produces a visible error. Archbase does not silently fall back to an ancestor, because that could expose the wrong architecture guidance.",
                callout: "Fix the invalid scope before relying on resolution or MCP output.",
            },
        ],
    },
    {
        slug: "local-customization",
        title: "Local customization",
        description: "Adapt catalog guidance without changing another scope or the public source.",
        group: "Core concepts",
        source: "docs/workspace.md",
        sections: [
            {
                id: "precedence",
                title: "Local copies take precedence",
                body: "A project-local customization overrides its catalog source during resolution.",
            },
            {
                id: "isolation",
                title: "Copies are independent",
                body: "Each scope keeps an independent pattern copy. Editing one copy does not automatically change the same pattern installed in another scope.",
            },
            {
                id: "agent-context",
                title: "Preserved for agents",
                body: "CLI inspection and read-only MCP tools expose the resolved local copy and preserve its source information.",
            },
        ],
    },
    {
        slug: "typed-markdown",
        title: "Typed Markdown examples",
        description: "Keep compilable-looking references outside normal project compilation entries.",
        group: "Core concepts",
        source: "docs/concepts.md",
        sections: [
            {
                id: "files",
                title: "Explicit example types",
                body: "Examples are Markdown documents named for the code they contain.",
                code: "Example.tsx.md\napplication.ts.md\ncontroller.cs.md",
                language: "text",
            },
            {
                id: "fences",
                title: "Language-aware code fences",
                body: "Each document contains code fences with an explicit language. This keeps references readable and prevents them from entering the normal build.",
            },
            {
                id: "mcp",
                title: "MCP extraction",
                body: "The MCP can return extracted code from typed Markdown while preserving the document origin.",
            },
        ],
    },
    {
        slug: "mcp-setup",
        title: "MCP setup",
        description: "Expose project architecture over a read-only stdio server.",
        group: "AI agent integrations",
        source: "docs/mcp.md",
        sections: [
            {
                id: "serve",
                title: "Start the server",
                body: "The MCP transport is stdio. Point it at the project root whose scopes and local copies it should resolve.",
                code: "arc mcp serve --project-root .",
                language: "bash",
            },
            {
                id: "configure",
                title: "Client configuration",
                body: "Use an absolute project path in the client configuration.",
                code: '{\n  "mcpServers": {\n    "archbase": {\n      "command": "arc",\n      "args": ["mcp", "serve", "--project-root", "/absolute/path/to/project"]\n    }\n  }\n}',
                language: "json",
            },
            {
                id: "safety",
                title: "Read-only by design",
                body: "The server is confined to the project root, rejects traversal and symlink escapes, reports errors visibly, and preserves local customizations. It cannot install patterns, export rules, or modify project code.",
            },
        ],
    },
    {
        slug: "mcp-tools",
        title: "MCP tools and boundaries",
        description: "The six read-only tools available to compatible agents.",
        group: "AI agent integrations",
        source: "docs/mcp.md",
        sections: [
            {
                id: "catalog-tools",
                title: "Catalog tools",
                body: "search_patterns searches the catalog. get_pattern returns a pattern manifest. get_pattern_files returns files declared by that manifest.",
            },
            {
                id: "project-tools",
                title: "Project tools",
                body: "resolve_pattern resolves the pattern for a path. get_scope_rules returns rules for the resolved context. list_project_scopes lists configured project scopes.",
            },
            {
                id: "boundaries",
                title: "Security boundaries",
                body: "Every project operation stays under the configured root. Traversal and symlink escapes are rejected, and an invalid scope remains an explicit error.",
                callout: "MCP is context delivery, not automation: it does not write files, install entries, or export agent instructions.",
            },
        ],
    },
];
export const catalogFamilies = [
    {
        slug: "nextjs",
        name: "Next.js",
        focus: "Modular or feature-oriented App Router organization, including explicit server/client boundaries.",
        rules: [
            "architecture/next-modular@1",
            "architecture/next-feature-app-router@1",
        ],
    },
    {
        slug: "react",
        name: "React",
        focus: "Vite projects organized around pages, components, hooks, and optional Tailwind conventions.",
        rules: [
            "architecture/react-vite-modular@1",
            "architecture/react-tailwind-modular@1",
        ],
    },
    {
        slug: "nestjs",
        name: "NestJS",
        focus: "Nest modules and services with explicit Prisma persistence responsibilities.",
        rules: ["architecture/nestjs-prisma-modular@1"],
    },
    {
        slug: "flutter",
        name: "Flutter",
        focus: "Flutter Provider with feature pages, API services, schemas, shared widgets and state ownership.",
        rules: ["architecture/flutter-provider-layered@1"],
    },
    {
        slug: "spring-boot",
        name: "Spring Boot",
        focus: "Layered Spring applications, with separate guidance for reactive MongoDB projects.",
        rules: [
            "architecture/spring-boot-layered@1",
            "architecture/spring-boot-reactive-mongo@1",
        ],
    },
    {
        slug: "dotnet",
        name: ".NET",
        focus: "Layered, EF Core, and multi-project variants with explicit project-reference direction.",
        rules: [
            "architecture/dotnet-layered@1",
            "architecture/dotnet-efcore-layered@1",
            "architecture/dotnet-multiproject@1",
        ],
    },
    {
        slug: "fastapi",
        name: "FastAPI",
        focus: "Python services split into routers, services, repositories, and schemas.",
        rules: ["architecture/python-fastapi-modular@1"],
    },
    {
        slug: "astro",
        name: "Astro",
        focus: "Content-driven sites with collections, layouts, and deliberate client hydration.",
        rules: ["architecture/astro-content-site@1"],
    },
    {
        slug: "discordjs",
        name: "Discord.js",
        focus: "Music bot organization with commands, events, services, and player boundaries.",
        rules: ["architecture/discordjs-music-bot@1"],
    },
];
const catalogPages: DocPage[] = catalogFamilies.map((family) => ({
    slug: `catalog-${family.slug}`,
    title: family.name,
    description: family.focus,
    group: "Catalog",
    source: "registry/rules/index.yaml",
    sections: [
        { id: "scope", title: "Scope and organization", body: family.focus },
        {
            id: "variants",
            title: "Available architecture rules",
            body: `Choose the variant that matches the project. Variants are alternatives, not cumulative requirements.`,
            code: family.rules.join("\n"),
            language: "text",
        },
        {
            id: "adopt",
            title: "Inspect and adopt",
            body: "Inspect the canonical rule and its related pattern identifiers before adding it to the project.",
            code: `arc rules inspect ${family.rules[0]}\narc rules add ${family.rules[0]} --format agents`,
            language: "bash",
        },
        {
            id: "limits",
            title: "Limits and local changes",
            body: "The rule is a reusable baseline. Keep framework-specific details current and use local customization when project constraints differ. A local copy takes precedence without changing the catalog.",
        },
    ],
}));
export const skills = [
    {
        slug: "architecture-starter",
        name: "Architecture Starter",
        family: "Foundation",
        version: "0.2.0",
        price: "Free",
        status: "Published",
        url: "https://www.agensi.io/skills/archbase-architecture-starter",
        summary: "Identifies stack and organization, explains applicable conventions, and suggests evidence-based next steps without requiring arc.",
    },
    {
        slug: "nextjs-auditor",
        name: "Next.js Architecture Auditor",
        family: "Auditor",
        version: "0.2.0",
        price: "US$9 one-time",
        status: "Published",
        url: "https://www.agensi.io/skills/archbase-next-js-architecture-auditor",
        summary: "Audits server/client boundaries, App Router placement, features, and dependency direction.",
    },
    {
        slug: "react-auditor",
        name: "React Auditor",
        family: "Auditor",
        version: "0.1.0",
        price: "US$9 one-time",
        status: "Published",
        url: "https://agensi.io/skills/archbase-react-architecture-auditor",
        summary: "Reviews pages, hooks, controllers, components, and modular boundaries.",
    },
    {
        slug: "nestjs-auditor",
        name: "NestJS Auditor",
        family: "Auditor",
        version: "0.1.0",
        price: "US$9 one-time",
        status: "Published",
        url: "https://agensi.io/skills/archbase-nestjs-architecture-auditor",
        summary: "Reviews modules, services, controllers, and Prisma persistence responsibilities.",
    },
    {
        slug: "flutter-auditor",
        name: "Flutter Auditor",
        family: "Auditor",
        version: "0.1.0",
        price: "US$9 one-time",
        status: "Published",
        url: "https://agensi.io/skills/archbase-flutter-architecture-auditor",
        summary: "Reviews Provider state ownership, feature pages, services, schemas and lifecycle boundaries.",
    },
    {
        slug: "spring-boot-auditor",
        name: "Spring Boot Auditor",
        family: "Auditor",
        version: "0.1.0",
        price: "US$9 one-time",
        status: "Published",
        url: "https://agensi.io/skills/archbase-spring-boot-architecture-auditor",
        summary: "Reviews persistence choices, controllers, services, repositories, and layer direction.",
    },
    {
        slug: "dotnet-auditor",
        name: ".NET Auditor",
        family: "Auditor",
        version: "0.1.0",
        price: "US$9 one-time",
        status: "Published",
        url: "https://agensi.io/skills/archbase-net-architecture-auditor",
        summary: "Reviews project references, composition roots, persistence, and multi-project boundaries.",
    },
    {
        slug: "fastapi-auditor",
        name: "FastAPI Auditor",
        family: "Auditor",
        version: "0.1.0",
        price: "US$9 one-time",
        status: "Published",
        url: "https://agensi.io/skills/archbase-fastapi-architecture-auditor",
        summary: "Reviews routers, services, repositories, schemas, and dependency direction.",
    },
    {
        slug: "astro-auditor",
        name: "Astro Auditor",
        family: "Auditor",
        version: "0.1.0",
        price: "US$9 one-time",
        status: "Published",
        url: "https://agensi.io/skills/archbase-astro-architecture-auditor",
        summary: "Reviews content collections, layouts, components, and client hydration boundaries.",
    },
    {
        slug: "migration-planner",
        name: "Refactoring & Migration Planner",
        family: "Planning",
        version: "0.1.0",
        price: "US$15 one-time",
        status: "Published",
        url: "https://agensi.io/skills/archbase-refactoring-migration-planner",
        summary: "Builds an incremental plan with dependencies, verification gates, and rollback boundaries. Initial references cover Next.js.",
    },
    {
        slug: "pr-architecture-guard",
        name: "PR Architecture Guard",
        family: "Review",
        version: "0.1.0",
        price: "US$15 one-time",
        status: "Published",
        url: "https://agensi.io/skills/archbase-pr-architecture-guard",
        summary: "Reviews BASE/HEAD diffs and separates regressions, fixes, and pre-existing debt without publishing or changing files.",
    },
] as const;
const skillPages: DocPage[] = skills.map((skill) => ({
    slug: skill.slug,
    title: skill.name,
    description: skill.summary,
    group: "Skills",
    source: skill.slug === "architecture-starter" ? "docs/skills.md" : "docs/premium-skills.md",
    sections: [
        { id: "purpose", title: "Purpose", body: skill.summary },
        {
            id: "availability",
            title: "Availability",
            body: `${skill.status} on Agensi · skill content ${skill.version} · ${skill.price}.`,
            links: [
                {
                    label: skill.price === "Free"
                        ? "Use free Starter on Agensi"
                        : "View on Agensi",
                    href: skill.url,
                },
            ],
        },
        {
            id: "output",
            title: skill.family === "Auditor" ? "Audit output" : "Workflow boundaries",
            body: skill.family === "Auditor"
                ? "The Markdown report records inspected scope, baseline, confirmed findings, suspicions, non-analyzable checks, file or line evidence, applicable rules, justified severity, ordered recommendations, and limitations. It does not assign an arbitrary score or modify code."
                : skill.slug === "migration-planner"
                    ? "The planner proposes an incremental sequence; it does not execute the migration or claim unrestricted support for every stack."
                    : skill.slug === "pr-architecture-guard"
                        ? "The guard does not publish reviews, approve or merge pull requests, or alter files automatically."
                        : "The starter provides orientation rather than an exhaustive architecture audit.",
        },
        {
            id: "evaluation",
            title: "Evaluation context",
            body: skill.slug === "architecture-starter" || skill.slug === "nextjs-auditor"
                ? "Evaluated locally with Codex CLI across 30 real scenarios and supplied snapshots. This is not framework certification or proof of delivery behavior in every agent."
                : skill.family === "Auditor"
                    ? "Evaluated locally across 30 real scenarios with Codex CLI and supplied snapshots. This does not certify other agents."
                    : skill.slug === "migration-planner"
                        ? "Evaluated locally across 6 real scenarios with initial Next.js references."
                        : "Evaluated locally across 12 diff and BASE/HEAD snapshot scenarios.",
        },
    ],
}));
const simple: Record<string, {
    description: string;
    sections: Section[];
    source?: string;
}> = {
    "workspace-layout": {
        description: "Understand canonical local storage for scopes, patterns, and rules.",
        source: "docs/workspace.md",
        sections: [
            {
                id: "root",
                title: "Project root",
                body: "Commands use the invocation directory as the project root unless --project-root is explicit.",
            },
            {
                id: "archbase-directory",
                title: "The .archbase directory",
                body: "Scopes are configured in .archbase/scopes.yaml and canonical architecture rules are stored under .archbase/rules/.",
            },
            {
                id: "ownership",
                title: "Keep generated indexes separate",
                body: "Agent-native indexes can be synchronized without duplicating the rule source or overwriting unrelated instructions.",
            },
        ],
    },
    "cli-overview": {
        description: "Use arc interactively or through explicit text commands.",
        source: "docs/getting-started.md",
        sections: [
            {
                id: "modes",
                title: "Interactive and plain modes",
                body: "arc opens the terminal UI. arc --plain keeps output textual for scripts and limited terminals.",
            },
            {
                id: "roots",
                title: "Project root behavior",
                body: "arc add and arc create start from the invocation directory unless --project-root is set.",
            },
            {
                id: "help",
                title: "Version-specific help",
                body: "Use the help bundled with your release for its exact flags. This portal does not document proposed arc audit or arc doctor commands as available.",
            },
        ],
    },
    "pattern-commands": {
        description: "Add, create, inspect, resolve, and remove pattern references.",
        source: "docs/getting-started.md",
        sections: [
            {
                id: "inspect-add",
                title: "Inspect and add",
                body: "Inspect a pinned catalog identifier, then add it to a project scope.",
                code: "arc inspect next/page@1234\narc add next/page@1234 .",
                language: "bash",
            },
            {
                id: "resolve",
                title: "Resolve",
                body: "Resolve reports the pattern active for a concrete path.",
                code: "arc resolve src/pages/Home.tsx",
                language: "bash",
            },
            {
                id: "roots",
                title: "Root selection",
                body: "Add and create default to the invocation directory. Pass --project-root only when the intended root is elsewhere.",
            },
        ],
    },
    "rule-commands": {
        description: "Inspect, add, export, and remove architecture rules.",
        source: "docs/rules.md",
        sections: [
            {
                id: "read",
                title: "List and inspect",
                body: "Review rule content and references before adoption.",
                code: "arc rules list\narc rules inspect architecture/next-modular@1",
                language: "bash",
            },
            {
                id: "export",
                title: "Add with an agent format",
                body: "Available native index formats include agents, claude, cursor, and copilot.",
                code: "arc rules add architecture/next-modular@1 --format agents\narc rules add architecture/next-modular@1 --format claude",
                language: "bash",
            },
            {
                id: "remove",
                title: "Remove",
                body: "Remove the canonical project rule and let Archbase synchronize formats that were activated.",
                code: "arc remove architecture/next-modular@1",
                language: "bash",
            },
        ],
    },
    "registry-configuration": {
        description: "Understand the public indexes that describe available patterns and rules.",
        source: "docs/registry.md",
        sections: [
            {
                id: "indexes",
                title: "Public indexes",
                body: "registry/index.yaml indexes patterns and registry/rules/index.yaml indexes architecture rules.",
            },
            {
                id: "pinned",
                title: "Pinned references",
                body: "Entry IDs and semantic versions are separate. Pin a registry revision to keep a reviewed catalog reproducible.",
            },
            {
                id: "local",
                title: "Local precedence",
                body: "Local copies override catalog content without mutating the public registry.",
            },
        ],
    },
    "updating-cli": {
        description: "Check and apply explicit CLI updates.",
        source: "docs/installation.md",
        sections: [
            {
                id: "commands",
                title: "Explicit update commands",
                body: "Check first or update directly.",
                code: "arc update --check\narc update",
                language: "bash",
            },
            {
                id: "behavior",
                title: "Updater behavior",
                body: "The updater verifies the download and replaces the executable in the current installation.",
            },
            {
                id: "limits",
                title: "What it does not change",
                body: "Updates do not modify projects or PATH, and Archbase does not run background updates.",
            },
        ],
    },
    "command-reference": {
        description: "A compact reference to stable, documented command examples.",
        source: "docs/getting-started.md",
        sections: [
            {
                id: "commands",
                title: "Documented commands",
                body: "These examples cover the stable workflows described in the public documentation.",
                code: "arc\narc version\narc update --check\narc add next/page@1234 .\narc resolve src/pages/Home.tsx\narc inspect next/page@1234\narc rules list\narc rules inspect architecture/next-modular@1\narc rules add architecture/next-modular@1 --format agents\narc remove architecture/next-modular@1",
                language: "bash",
            },
            {
                id: "exact-flags",
                title: "Exact flags",
                body: "Consult arc help in the installed release for the complete flag reference. Fixed identifiers in examples can represent historical pinned versions.",
            },
        ],
    },
    catalog: {
        description: "Fourteen public architecture rules across nine stack families.",
        source: "registry/rules/index.yaml",
        sections: [
            {
                id: "scope",
                title: "Catalog scope",
                body: "The official catalog contains patterns and architecture references—not complete applications or product boilerplates.",
            },
            {
                id: "selection",
                title: "Choose one applicable variant",
                body: "Variants in a family are not cumulative. Inspect responsibilities, related patterns, and limits before adoption.",
            },
            {
                id: "source",
                title: "Canonical indexes",
                body: "The public registry indexes are the source of truth for identifiers and available versions.",
            },
        ],
    },
    "agent-integrations": {
        description: "Export rules into native agent indexes or expose read-only context through MCP.",
        source: "docs/rules.md",
        sections: [
            {
                id: "exports",
                title: "Native rule indexes",
                body: "Canonical rules live once in .archbase/rules/. Archbase maintains indexes in AGENTS.md, CLAUDE.md, .cursor/rules/archbase.mdc, and .github/copilot-instructions.md.",
            },
            {
                id: "formats",
                title: "Formats",
                body: "Activate a format when adding a rule.",
                code: "--format agents\n--format claude\n--format cursor\n--format copilot",
                language: "text",
            },
            {
                id: "separation",
                title: "Exports are not skill evaluations",
                body: "The CLI can export rule indexes for an agent, but that does not imply every skill has been behaviorally evaluated in every agent.",
                callout: "New rule records synchronize formats already activated while preserving external instructions.",
            },
        ],
    },
    skills: {
        description: "Compare open-source orientation with premium audit, planning, and review workflows.",
        source: "docs/premium-skills.md",
        sections: [
            {
                id: "open-source",
                title: "Public and MIT",
                body: "The Go CLI, MCP, official catalog, patterns, rules, core fixes, Architecture Starter, documentation, and installation remain public under MIT.",
            },
            {
                id: "premium",
                title: "Premium workflows",
                body: "Premium adds original proprietary workflows and templates. Embedded public references retain their MIT license and attribution.",
            },
            {
                id: "availability",
                title: "Verified availability",
                body: "All eleven skills are approved and published on Agensi. Starter is free; eight auditors cost US$9 each; Planner and PR Guard cost US$15 each, as one-time purchases. Verified November 4, 2026.",
            },
        ],
    },
    schemas: {
        description: "Reference the public data contracts used by manifests, scopes, and rules.",
        source: "docs/schemas.md",
        sections: [
            {
                id: "purpose",
                title: "Machine-readable contracts",
                body: "Schemas make registry entries, local workspace files, and tool output explicit and validateable.",
            },
            {
                id: "compatibility",
                title: "Version-aware data",
                body: "Use the schemas shipped with the relevant Archbase release and validate changes before publishing registry content.",
            },
        ],
    },
    privacy: {
        description: "Understand what local commands and integrations can access.",
        source: "docs/privacy.md",
        sections: [
            {
                id: "local-first",
                title: "Local-first operation",
                body: "Project scopes, local patterns, and canonical rules remain in the project workspace.",
            },
            {
                id: "mcp",
                title: "MCP boundary",
                body: "The MCP is read-only and confined to the configured project root. It rejects traversal and symlink escapes.",
            },
            {
                id: "tracking",
                title: "No site tracking",
                body: "This documentation portal does not add analytics or tracking.",
            },
        ],
    },
    "release-integrity": {
        description: "Verify release assets and understand platform support.",
        source: "docs/code-signing-policy.md",
        sections: [
            {
                id: "checksums",
                title: "Verify checksums",
                body: "Compare downloaded assets with checksums published on the release before executing them.",
            },
            {
                id: "platforms",
                title: "Platform channels",
                body: "Linux is the official installation channel. macOS and Windows are unsigned preview builds.",
            },
            {
                id: "claims",
                title: "No signing claim",
                body: "Archbase does not claim that preview binaries are signed or certified.",
            },
        ],
    },
    contributing: {
        description: "Contribute changes to the CLI, catalog, and documentation.",
        source: "docs/development.md",
        sections: [
            {
                id: "start",
                title: "Start with the public repository",
                body: "Open an issue or pull request with a focused change and evidence for the behavior or convention being proposed.",
            },
            {
                id: "registry",
                title: "Catalog changes",
                body: "Update the relevant public index and canonical documents together. Keep identifiers versioned and examples typed.",
            },
        ],
    },
    releases: {
        description: "Find the latest published CLI assets and release notes.",
        source: "README.md",
        sections: [
            {
                id: "latest",
                title: "Latest release",
                body: "Use the public GitHub releases page for the current stable version, notes, assets, and checksums.",
            },
            {
                id: "historical",
                title: "Pinned historical examples",
                body: "Some documentation examples intentionally pin historical pattern or rule versions. A pinned example is not a claim about a new stable CLI version.",
            },
        ],
    },
};
const integrations: Record<string, [
    string,
    string
]> = {
    "agents-md": ["AGENTS.md", "agents"],
    claude: ["CLAUDE.md", "claude"],
    cursor: [".cursor/rules/archbase.mdc", "cursor"],
    "github-copilot": [".github/copilot-instructions.md", "copilot"],
};
const generatedPages: DocPage[] = [];
for (const [slug, item] of Object.entries(simple)) {
    const title = navGroups.flatMap((g) => g.pages).find((p) => p.slug === slug)?.title ??
        slug;
    const group = navGroups.find((g) => g.pages.some((p) => p.slug === slug))?.title ??
        "Reference and project";
    generatedPages.push({
        slug,
        title,
        description: item.description,
        sections: item.sections,
        group,
        source: item.source,
    });
}
for (const [slug, [file, format]] of Object.entries(integrations)) {
    generatedPages.push({
        slug,
        title: navGroups.flatMap((g) => g.pages).find((p) => p.slug === slug)?.title ??
            slug,
        description: `Expose canonical Archbase rules through ${file}.`,
        group: "AI agent integrations",
        source: "docs/rules.md",
        sections: [
            {
                id: "destination",
                title: "Native destination",
                body: `Archbase writes its managed rule index to ${file} while canonical rule documents remain under .archbase/rules/.`,
            },
            {
                id: "activate",
                title: "Activate the format",
                body: "Select the format while adding a reviewed rule.",
                code: `arc rules add architecture/next-modular@1 --format ${format}`,
                language: "bash",
            },
            {
                id: "preservation",
                title: "Synchronization and preservation",
                body: "Later rule additions synchronize formats that are already active. Instructions outside the managed Archbase section are preserved.",
            },
        ],
    });
}
const originalPages: DocPage[] = [
    ...basePages,
    ...generatedPages,
    ...catalogPages,
    ...skillPages,
];
const originalMap = new Map(originalPages.map((page) => [page.slug, page]));
const merge = (slug: string, title: string, group: string, source: string, inputs: string[]): DocPage => ({
    slug,
    title,
    group,
    source,
    description: originalMap.get(inputs[0])!.description,
    sections: inputs.flatMap((input) => originalMap
        .get(input)!
        .sections.map((item) => ({ ...item, id: `${input}-${item.id}` }))),
});
const concepts = merge("concepts", "Patterns, rules & scopes", "Guides", "docs/concepts.md", [
    "patterns",
    "architecture-rules",
    "scopes-and-resolution",
    "local-customization",
    "workspace-layout",
    "typed-markdown",
]);
concepts.sections = [
    {
        id: "patterns",
        title: "Patterns describe one code type",
        body: "Patterns provide a validated manifest and focused examples for pages, hooks, controllers and other code types. They are references, not complete feature generators. PatternID next/page@1234 is separate from its manifest semantic version.",
        code: "arc inspect next/page@1234\narc add next/page@1234 .",
        language: "bash",
    },
    {
        id: "rules",
        title: "Rules define architecture",
        body: "Rules associate paths with PatternIDs and define responsibilities and dependency direction. Canonical documents live once under .archbase/rules/; native agent indexes reference them. Choose the compatible adopted architecture variant, not every variant together.",
    },
    {
        id: "scopes",
        title: "The most specific scope wins",
        body: "The root .archbase/scopes.yaml selects patterns by directory. src/pages/admin overrides src/pages. Targets need not exist yet. Invalid scopes or a selected invalid bundle fail visibly, without fallback to an ancestor.",
        code: "arc resolve src/pages/admin/Dashboard.tsx",
        language: "bash",
    },
    {
        id: "customization",
        title: "Customize independent local copies",
        body: "Local pattern copies take precedence over their registry sources. Each scope is independent. Use the root printed by resolve/inspect to locate examples. Activating another pattern preserves stored bundles and refuses collisions.",
    },
    {
        id: "workspace",
        title: "Workspace and typed examples",
        body: "All metadata lives at the project root. Source examples are stored as typed Markdown, with exactly one language-aware fence, so reference code does not enter compiler inputs. MCP returns extracted source code. Preserve the language and fence when editing.",
        code: ".archbase/\n  scopes.yaml\n  rules/architecture-next-modular-1.md\n  patterns/_root/next--page--1234/\n    manifest.yaml\n    Example/Example.tsx.md",
        language: "text",
        links: [
            { label: "Workspace contract", href: sourceRoot + "docs/workspace.md" },
        ],
    },
];
const cli = merge("cli", "CLI & registries", "Guides", "docs/terminal-ui.md", [
    "terminal-ui",
    "pattern-commands",
    "rule-commands",
    "registry-configuration",
]);
cli.sections = [
    {
        id: "terminal",
        title: "Interactive or plain",
        body: "Run arc in a compatible terminal. Ctrl+P opens the palette; Esc goes back. --plain uses textual output; redirected output stays plain. TERM=dumb disables the UI; NO_COLOR removes colors while keeping navigation.",
        code: "arc\narc --plain\narc --help",
        language: "bash",
    },
    {
        id: "patterns",
        title: "Inspect, install and customize",
        body: "add/create use the invocation directory as root unless --project-root selects another. They do not infer a Git root. create derives a local independent pattern.",
        code: "arc inspect next/page@1234\narc add next/page@1234 .\narc create pages-standard src/pages/admin --from next/page@1234\narc resolve src/pages/admin/Dashboard.tsx",
        language: "bash",
    },
    {
        id: "rules",
        title: "Manage architecture rules",
        body: "Review a rule before activation. Later changes synchronize all active native formats and preserve external instructions. remove removes owned rule exports, not patterns or scopes. Changed owned documents require explicit --overwrite.",
        code: "arc rules list\narc rules inspect architecture/next-modular@1\narc rules add architecture/next-modular@1 --format agents\narc remove architecture/next-modular@1\narc remove --all",
        language: "bash",
    },
    {
        id: "registry",
        title: "Embedded and public Git catalogs",
        body: "The embedded catalog works offline. Explicit public Git registries can refresh a local cache. Pin --registry-ref to a reviewed tag/commit for reproducibility. Invalid registries fail visibly. Authentication remains outside the current scope.",
        code: "arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.8.0 --registry-subdir registry rules list",
        language: "bash",
        links: [
            {
                label: "Registry and caching guide",
                href: sourceRoot + "docs/registry.md",
            },
        ],
    },
    {
        id: "help",
        title: "Version-specific help",
        body: "Use --help in your installed release for exact flags. arc audit and arc doctor remain proposals, not available commands. Updates are covered under Installation & updates.",
    },
];
const agents = merge("agents", "Agent instructions", "Guides", "docs/rules.md", ["agent-integrations"]);
agents.sections = [
    {
        id: "formats",
        title: "Activate your native format",
        body: "Each command is an alternative. Canonical documents stay in .archbase/rules/; native files receive managed indexes.",
        code: "arc rules add architecture/next-modular@1 --format agents\narc rules add architecture/next-modular@1 --format claude\narc rules add architecture/next-modular@1 --format cursor\narc rules add architecture/next-modular@1 --format copilot",
        language: "bash",
    },
    {
        id: "destinations",
        title: "One source, four native entry points",
        body: "Managed indexes live in AGENTS.md, CLAUDE.md, .cursor/rules/archbase.mdc and .github/copilot-instructions.md. Later changes synchronize already active formats. External instructions are preserved; Cursor requires valid alwaysApply frontmatter.",
    },
    {
        id: "safety",
        title: "Preservation and compatibility",
        body: "Identical owned documents are reused; changed owned documents require --overwrite. Foreign files and collisions are refused. Removal preserves external content and native entry files. Native rule export support does not certify every skill in that host.",
    },
];
const installation = originalMap.get("installation")!;
installation.title = "Installation & updates";
installation.sections.push({
    id: "update",
    title: "Update the installed CLI",
    body: "Explicit updates verify release checksum and executable version and replace only the invoked installation. No downgrade, automatic elevation, project/PATH changes or background checks. Older binaries without update need one installer reinstall. Restart running MCP processes after updating.",
    code: "arc update --check\narc update\narc version",
    language: "bash",
    links: [
        {
            label: "Full installation, preview installers and removal",
            href: sourceRoot + "docs/installation.md",
        },
    ],
});
installation.sections[1].callout =
    "macOS and Windows remain unsigned previews. Verify checksums and respect OS protections; do not bypass them.";
const mcp = merge("mcp-setup", "Read-only MCP", "Guides", "docs/mcp.md", [
    "mcp-setup",
    "mcp-tools",
]);
mcp.sections = mcp.sections.filter((section) => section.id !== "mcp-tools-boundaries");
const project = merge("project", "Project & privacy", "Reference", "docs/privacy.md", ["privacy", "release-integrity", "contributing", "releases"]);
project.sections = [
    {
        id: "privacy-local-first",
        title: "Local data and privacy",
        body: "Archbase has no accounts, telemetry, analytics, crash-report collection or background update requests. It contacts the network only for explicit installer/update or configured registry operations. Hosting providers have their own logs and policies. Your agent/provider processes supplied context separately.",
        links: [{ label: "Privacy policy", href: sourceRoot + "docs/privacy.md" }],
    },
    {
        id: "releases",
        title: "Release integrity and platforms",
        body: "Linux is the official distribution; macOS and Windows are unsigned previews. Verify release checksums; these establish artifact integrity, not a trusted code-signing identity. Signing integration remains inactive. Respect OS security protections.",
        links: [
            {
                label: "Latest stable release",
                href: "https://github.com/EnzoCaetano015/Archbase/releases/latest",
            },
            {
                label: "Signing policy",
                href: sourceRoot + "docs/code-signing-policy.md",
            },
        ],
    },
    {
        id: "contributing",
        title: "Contribute to the public core and catalog",
        body: "CLI, MCP, official patterns/rules, fixes and Starter stay MIT. Preserve local precedence, required-file validation and agent-neutral rule documents. See the development guide for checks and release gates. Architecture diagnostics proposals do not imply implemented CLI commands.",
        links: [
            { label: "Development guide", href: sourceRoot + "docs/development.md" },
            {
                label: "Report an issue",
                href: "https://github.com/EnzoCaetano015/Archbase/issues",
            },
        ],
    },
];
export const pages: DocPage[] = [
    originalMap.get("introduction")!,
    originalMap.get("quick-start")!,
    installation,
    concepts,
    cli,
    agents,
    mcp,
    originalMap.get("catalog")!,
    ...catalogPages,
    originalMap.get("skills")!,
    ...skillPages,
    originalMap.get("schemas")!,
    project,
];
export const docAliases: Record<string, string> = {
    "terminal-ui": "cli#terminal",
    patterns: "concepts#patterns",
    "architecture-rules": "concepts#rules",
    "scopes-and-resolution": "concepts#scopes",
    "local-customization": "concepts#customization",
    "workspace-layout": "concepts#workspace",
    "typed-markdown": "concepts#workspace",
    "cli-overview": "cli",
    "pattern-commands": "cli#patterns",
    "rule-commands": "cli#rules",
    "registry-configuration": "cli#registry",
    "updating-cli": "installation#update",
    "command-reference": "cli",
    "agent-integrations": "agents",
    "agents-md": "agents#formats",
    claude: "agents#formats",
    cursor: "agents#formats",
    "github-copilot": "agents#formats",
    "mcp-tools": "mcp-setup",
    privacy: "project#privacy-local-first",
    "release-integrity": "project",
    releases: "project",
    contributing: "project",
    scopes: "concepts#scopes",
    mcp: "mcp-setup",
};
export const pageMap = new Map(pages.map((page) => [page.slug, page]));
export const catalogRules = catalogDocuments;
export const sourceUrl = (page: DocPage) => `${sourceRoot}${page.source ?? "README.md"}`;
