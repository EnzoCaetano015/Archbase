import { useEffect, useMemo, useRef, useState } from "react";
import { catalogFamilies, catalogRules, docAliases, navGroups, pageMap, pages, skills, sourceUrl, type DocPage, type Section, } from "./content";
const GITHUB = "https://github.com/EnzoCaetano015/Archbase";
const INSTALL = "curl -fsSL https://archbase.caetanodev.com/install.sh | sh";
type IconName = "search" | "sun" | "moon" | "menu" | "close" | "github" | "copy" | "check" | "arrow" | "external" | "terminal" | "book" | "layers";
function Icon({ name, size = 18 }: {
    name: IconName;
    size?: number;
}) {
    const common = {
        width: size,
        height: size,
        viewBox: "0 0 24 24",
        fill: "none",
        stroke: "currentColor",
        strokeWidth: 1.8,
        strokeLinecap: "round" as const,
        strokeLinejoin: "round" as const,
        "aria-hidden": true,
    };
    const paths: Record<IconName, React.ReactNode> = {
        search: (<>
        <circle cx="11" cy="11" r="7"/>
        <path d="m20 20-4-4"/>
      </>),
        sun: (<>
        <circle cx="12" cy="12" r="4"/>
        <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>
      </>),
        moon: <path d="M20 15.5A8.5 8.5 0 0 1 8.5 4 8.5 8.5 0 1 0 20 15.5Z"/>,
        menu: (<>
        <path d="M4 7h16M4 12h16M4 17h16"/>
      </>),
        close: (<>
        <path d="m6 6 12 12M18 6 6 18"/>
      </>),
        github: (<path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3.3-.4 6.8-1.6 6.8-7A5.5 5.5 0 0 0 19.3 3 5.1 5.1 0 0 0 19.2 0S18 0 15 1.7a15.4 15.4 0 0 0-8 0C4 0 2.8 0 2.8 0a5.1 5.1 0 0 0-.1 3A5.5 5.5 0 0 0 1.2 7.5c0 5.4 3.5 6.6 6.8 7A4.8 4.8 0 0 0 7 18v4M7 19c-3 .9-3-1.5-4-2"/>),
        copy: (<>
        <rect x="8" y="8" width="12" height="12" rx="2"/>
        <path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2"/>
      </>),
        check: <path d="m5 12 4 4L19 6"/>,
        arrow: (<>
        <path d="M5 12h14M13 6l6 6-6 6"/>
      </>),
        external: (<>
        <path d="M14 4h6v6M20 4l-9 9"/>
        <path d="M18 13v6a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h6"/>
      </>),
        terminal: (<>
        <path d="m5 7 4 4-4 4M12 17h7"/>
        <rect x="2" y="3" width="20" height="18" rx="2"/>
      </>),
        book: (<>
        <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
        <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2Z"/>
      </>),
        layers: (<>
        <path d="m12 2 9 5-9 5-9-5 9-5Z"/>
        <path d="m3 12 9 5 9-5M3 17l9 5 9-5"/>
      </>),
    };
    return <svg {...common}>{paths[name]}</svg>;
}
function Logo() {
    return (<img className="logo-mark" src="/assets/archbase-mark.svg" alt="" aria-hidden="true"/>);
}
function navigate(path: string) {
    window.history.pushState({}, "", path);
    window.dispatchEvent(new PopStateEvent("popstate"));
    const hash = new URL(path, window.location.origin).hash.slice(1);
    if (hash)
        requestAnimationFrame(() => document.getElementById(decodeURIComponent(hash))?.scrollIntoView());
    else
        window.scrollTo({ top: 0 });
}
function Link({ href, children, className = "", onClick, ...rest }: {
    href: string;
    children: React.ReactNode;
    className?: string;
    onClick?: () => void;
    "aria-label"?: string;
    "aria-current"?: "page";
}) {
    const external = href.startsWith("http");
    return (<a {...rest} href={href} className={className} target={external ? "_blank" : undefined} rel={external ? "noreferrer" : undefined} onClick={(event) => {
            onClick?.();
            if (!external &&
                !event.defaultPrevented &&
                event.button === 0 &&
                !event.shiftKey &&
                !event.altKey &&
                !event.metaKey &&
                !event.ctrlKey) {
                event.preventDefault();
                navigate(href);
            }
        }}>
      {children}
    </a>);
}
function CodeBlock({ code, language = "text", }: {
    code: string;
    language?: string;
}) {
    const [copied, setCopied] = useState(false);
    return (<div className="code-block">
      <div className="code-head">
        <span>{language}</span>
        <button onClick={() => {
            navigator.clipboard.writeText(code);
            setCopied(true);
            window.setTimeout(() => setCopied(false), 1600);
        }} aria-label="Copy code">
          <Icon name={copied ? "check" : "copy"} size={15}/>
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
      <pre>
        <code>{code}</code>
      </pre>
    </div>);
}
function Header({ onMenu, onSearch, theme, setTheme, }: {
    onMenu: () => void;
    onSearch: () => void;
    theme: string;
    setTheme: (theme: string) => void;
}) {
    return (<header className="site-header">
      <div className="header-inner">
        <button className="icon-button mobile-menu" onClick={onMenu} aria-label="Open navigation">
          <Icon name="menu"/>
        </button>
        <Link href="/" className="brand" aria-label="Archbase home">
          <Logo />
          <span>Archbase</span>
        </Link>
        <nav className="top-nav" aria-label="Main navigation">
          <Link href="/docs/introduction">Documentation</Link>
          <Link href="/docs/catalog">Catalog</Link>
          <Link href="/skills/">Skills</Link>
          <a href={GITHUB} target="_blank" rel="noreferrer">
            GitHub
          </a>
        </nav>
        <div className="header-actions">
          <button className="search-trigger" onClick={onSearch} aria-label="Search documentation">
            <Icon name="search" size={16}/>
            <span>Search docs</span>
            <kbd>⌘ K</kbd>
          </button>
          <button className="icon-button" onClick={() => setTheme(theme === "dark" ? "light" : "dark")} aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} theme`}>
            <Icon name={theme === "dark" ? "sun" : "moon"}/>
          </button>
          <a className="icon-button github-mobile" href={GITHUB} target="_blank" rel="noreferrer" aria-label="GitHub">
            <Icon name="github"/>
          </a>
        </div>
      </div>
    </header>);
}
function SearchDialog({ open, close }: {
    open: boolean;
    close: () => void;
}) {
    const [query, setQuery] = useState("");
    const input = useRef<HTMLInputElement>(null);
    const dialog = useRef<HTMLDivElement>(null);
    useEffect(() => {
        if (!open)
            return;
        const previous = document.activeElement as HTMLElement | null;
        setQuery("");
        input.current?.focus();
        const trap = (event: KeyboardEvent) => {
            if (event.key !== "Tab")
                return;
            const focusable = dialog.current?.querySelectorAll<HTMLElement>("a[href], button, input");
            if (!focusable?.length)
                return;
            const first = focusable[0], last = focusable[focusable.length - 1];
            if (event.shiftKey && document.activeElement === first) {
                event.preventDefault();
                last.focus();
            }
            else if (!event.shiftKey && document.activeElement === last) {
                event.preventDefault();
                first.focus();
            }
        };
        document.addEventListener("keydown", trap);
        return () => {
            document.removeEventListener("keydown", trap);
            previous?.focus();
        };
    }, [open]);
    useEffect(() => {
        const listener = (event: KeyboardEvent) => {
            if (event.key === "Escape")
                close();
            if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
                event.preventDefault();
                open ? close() : undefined;
            }
        };
        window.addEventListener("keydown", listener);
        return () => window.removeEventListener("keydown", listener);
    }, [open, close]);
    const results = useMemo(() => {
        const q = query.trim().toLowerCase();
        if (!q)
            return pages.slice(0, 7).map((page) => ({ page, section: undefined }));
        return pages
            .flatMap<{
            page: DocPage;
            section?: Section;
        }>((page) => {
            const pageMatch = `${page.title} ${page.description} ${page.group}`
                .toLowerCase()
                .includes(q);
            const sections = page.sections.filter((section) => `${section.title} ${section.body} ${section.code ?? ""}`
                .toLowerCase()
                .includes(q));
            if (sections.length)
                return sections.map((section) => ({ page, section }));
            if (pageMatch)
                return [{ page, section: undefined }];
            const family = catalogFamilies.find((family) => page.slug === `catalog-${family.slug}`);
            if (family &&
                catalogRules.some((rule) => family.rules.includes(rule.id) &&
                    `${rule.id} ${rule.name} ${rule.examples.map((example) => `${example.patternId} ${example.source} ${example.code}`).join(" ")}`
                        .toLowerCase()
                        .includes(q))) {
                return [
                    {
                        page,
                        section: {
                            id: "canonical-examples",
                            title: `${page.title} code examples`,
                            body: "Canonical pattern source and architecture mappings.",
                        },
                    },
                ];
            }
            return [];
        })
            .slice(0, 10);
    }, [query]);
    if (!open)
        return null;
    return (<div className="modal-backdrop" role="presentation" onMouseDown={close}>
      <div ref={dialog} className="search-modal" role="dialog" aria-modal="true" aria-label="Search documentation" onMouseDown={(e) => e.stopPropagation()}>
        <div className="search-input">
          <Icon name="search"/>
          <input ref={input} value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search commands, concepts, and pages…"/>
          <kbd>Esc</kbd>
        </div>
        <div className="search-results">
          <div className="result-label">
            {query ? `${results.length} results` : "Suggested pages"}
          </div>
          {results.map(({ page, section }, index) => (<Link key={`${page.slug}-${section?.id ?? index}`} href={`/docs/${page.slug}${section ? `#${section.id}` : ""}`} className="search-result" onClick={close}>
              <Icon name={page.group === "CLI"
                ? "terminal"
                : page.group === "Catalog"
                    ? "layers"
                    : "book"} size={18}/>
              <span>
                <strong>{section?.title ?? page.title}</strong>
                <small>
                  {section
                ? `${page.title} · ${section.body.slice(0, 100)}…`
                : page.description}
                </small>
              </span>
              <Icon name="arrow" size={16}/>
            </Link>))}
          {!results.length && (<div className="empty-search">
              No matching documentation. Try a command, stack, or concept.
            </div>)}
        </div>
      </div>
    </div>);
}
function Sidebar({ current, open, close, }: {
    current?: string;
    open: boolean;
    close: () => void;
}) {
    return (<>
      {open && (<button className="drawer-backdrop" onClick={close} aria-label="Close navigation"/>)}
      <aside className={`sidebar ${open ? "open" : ""}`}>
        <div className="drawer-head">
          <span>Documentation</span>
          <button className="icon-button" onClick={close} aria-label="Close navigation">
            <Icon name="close"/>
          </button>
        </div>
        <nav aria-label="Documentation navigation">
          {navGroups.map((group) => (<details key={group.title} open>
              <summary>
                {group.title}
                <span>⌄</span>
              </summary>
              <div className="nav-pages">
                {group.pages.map((page) => (<Link onClick={close} key={page.slug} href={`/docs/${page.slug}`} className={current === page.slug ? "active" : ""} aria-current={current === page.slug ? "page" : undefined}>
                    {page.title}
                  </Link>))}
              </div>
            </details>))}
        </nav>
        <div className="sidebar-foot">
          <span className="status-dot"/>
          Open source · MIT
        </div>
      </aside>
    </>);
}
function TableOfContents({ page }: {
    page: DocPage;
}) {
    const [active, setActive] = useState(page.sections[0]?.id);
    useEffect(() => {
        const observer = new IntersectionObserver((entries) => {
            const visible = entries
                .filter((entry) => entry.isIntersecting)
                .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
            if (visible[0]?.target.id)
                setActive(visible[0].target.id);
        }, { rootMargin: "-90px 0px -65% 0px" });
        [...page.sections, ...(page.slug.startsWith("catalog-") ? [{id: "canonical-examples"}] : [])].forEach((section) => {
            const node = document.getElementById(section.id);
            if (node)
                observer.observe(node);
        });
        return () => observer.disconnect();
    }, [page]);
    return (<aside className="toc">
      <p>On this page</p>
      <nav>
        {page.sections.map((section) => (<a key={section.id} className={active === section.id ? "active" : ""} href={`#${section.id}`}>
            {section.title}
          </a>))}
        {page.slug.startsWith("catalog-") && (<a className={active === "canonical-examples" ? "active" : ""} href="#canonical-examples">Code examples</a>)}
      </nav>
      <a className="edit-link" href={sourceUrl(page)} target="_blank" rel="noreferrer">
        <Icon name="github" size={15}/>
        Edit this page
      </a>
    </aside>);
}
function PrevNext({ page }: {
    page: DocPage;
}) {
    if (page.group === "Skills" && page.slug !== "skills")
        return (<div className="prev-next">
        <Link href="/skills/">← All skills</Link>
      </div>);
    const flat = navGroups.flatMap((group) => group.pages);
    const index = flat.findIndex((item) => item.slug === page.slug);
    const prev = flat[index - 1];
    const next = flat[index + 1];
    return (<div className="prev-next">
      {prev ? (<Link href={`/docs/${prev.slug}`}>
          <small>Previous</small>
          <span>← {prev.title}</span>
        </Link>) : (<span />)}
      {next ? (<Link href={`/docs/${next.slug}`}>
          <small>Next</small>
          <span>{next.title} →</span>
        </Link>) : (<span />)}
    </div>);
}
function DocContent({ page }: {
    page: DocPage;
}) {
    return (<article className="doc-article">
      <div className="breadcrumbs">
        <Link href="/">Home</Link>
        <span>/</span>
        <span>{page.group}</span>
        <span>/</span>
        <strong>{page.title}</strong>
      </div>
      <div className="page-kicker">{page.group}</div>
      <h1>{page.title}</h1>
      <p className="lead">{page.description}</p>
      <div className="page-rule"/>
      {page.sections.map((section) => (<section id={section.id} key={section.id} className="doc-section">
          <a className="heading-anchor" href={`#${section.id}`} aria-label={`Link to ${section.title}`}>
            #
          </a>
          <h2>{section.title}</h2>
          <p>{section.body}</p>
          {section.links && (<div className="source-links">
              {section.links.map((link) => (<a key={link.href} href={link.href} target="_blank" rel="noreferrer">
                  {link.label} <Icon name="external" size={13}/>
                </a>))}
            </div>)}
          {section.callout && (<div className="callout">
              <span>Note</span>
              <p>{section.callout}</p>
            </div>)}
          {section.code && (<CodeBlock code={section.code} language={section.language}/>)}
        </section>))}
      {page.slug.startsWith("catalog-") && <CatalogExamples slug={page.slug}/>}
      {page.group === "Catalog" && page.slug === "catalog" && (<CatalogGrid compact/>)}
      {page.slug === "skills" && (<div className="inline-note">
          Visit the <Link href="/skills/">Skills directory</Link> to compare
          availability and pricing.
        </div>)}
      <div className="doc-footer">
        <a href={sourceUrl(page)} target="_blank" rel="noreferrer">
          <Icon name="github" size={16}/>
          Edit this page on GitHub
          <Icon name="external" size={13}/>
        </a>
        <span>Last reviewed November 4, 2026</span>
      </div>
      <PrevNext page={page}/>
    </article>);
}
function CatalogExamples({ slug }: {
    slug: string;
}) {
    const family = catalogFamilies.find((item) => `catalog-${item.slug}` === slug)!;
    return (<div className="catalog-examples">
      <h2 id="canonical-examples">Code and architecture examples</h2>
      <p>
        These are the actual public catalog references, not generated
        application code. Expand a file to see the complete example.
        Placeholders describe how your agent should adapt each pattern; local
        conventions take precedence.
      </p>
      {catalogRules
            .filter((rule) => family.rules.includes(rule.id))
            .map((rule) => (<section key={rule.id} className="rule-example">
            <div className="variant-heading">
              <h3>{rule.name}</h3>
              <code>{rule.id}</code>
            </div>
            <p>{rule.description}</p>
            <div className="source-links">
              <a href={rule.url} target="_blank" rel="noreferrer">
                Canonical rule <Icon name="external" size={13}/>
              </a>
              <a href={rule.notesUrl} target="_blank" rel="noreferrer">
                Reference notes <Icon name="external" size={13}/>
              </a>
            </div>
            <h4>Where each responsibility belongs</h4>
            <div className="mapping-table">
              <table>
                <thead>
                  <tr>
                    <th>Project path</th>
                    <th>Pattern</th>
                  </tr>
                </thead>
                <tbody>
                  {rule.scopes.map((scope, index) => (<tr key={index}>
                      <td>
                        <code>{scope.path}</code>
                      </td>
                      <td>
                        <code>{scope.pattern}</code>
                      </td>
                    </tr>))}
                </tbody>
              </table>
            </div>
            <details className="architecture-details">
              <summary>
                Architecture constraints ({rule.restrictions.length})
              </summary>
              <ul>
                {rule.restrictions.map((text) => (<li key={text}>{text}</li>))}
              </ul>
            </details>
            {rule.examples.map((example, index) => (<details className="example-file" key={example.source} open={index === 0}>
                <summary>
                  <span>{example.source.split("/").at(-1)}</span>
                  <small>{example.patternId}</small>
                </summary>
                <div className="example-meta">
                  <p>
                    {example.name} · destination:{" "}
                    <code>{example.destination}</code>
                  </p>
                  <div className="source-links">
                    <a href={example.url} target="_blank" rel="noreferrer">
                      View original example <Icon name="external" size={13}/>
                    </a>
                    <a href={example.manifestUrl} target="_blank" rel="noreferrer">
                      Pattern manifest <Icon name="external" size={13}/>
                    </a>
                  </div>
                </div>
                <CodeBlock code={example.code} language={example.language}/>
              </details>))}
          </section>))}
      <p className="catalog-attribution">
        Catalog examples and rules are MIT licensed. Source text is generated
        directly from the public registry during build.
      </p>
    </div>);
}
function DocsPage({ slug, menuOpen, closeMenu, }: {
    slug: string;
    menuOpen: boolean;
    closeMenu: () => void;
}) {
    const page = pageMap.get(slug)!;
    useEffect(() => {
        document.title = `${page.title} · Archbase Docs`;
    }, [page]);
    return (<div className="docs-shell">
      <Sidebar current={page.slug} open={menuOpen} close={closeMenu}/>
      <main id="main" className="doc-main">
        <DocContent page={page}/>
      </main>
      <TableOfContents page={page}/>
    </div>);
}
function CatalogGrid({ compact = false }: {
    compact?: boolean;
}) {
    return (<div className={`catalog-grid ${compact ? "compact" : ""}`}>
      {catalogFamilies.map((family) => (<Link href={`/docs/catalog-${family.slug}`} className="catalog-card" key={family.slug}>
          <div className="catalog-card-top">
            <span className="stack-mark">{family.name.slice(0, 2)}</span>
            <span>
              {family.rules.length}{" "}
              {family.rules.length === 1 ? "rule" : "rules"}
            </span>
          </div>
          <h3>{family.name}</h3>
          <p>{family.focus}</p>
          <div className="rule-id">{family.rules[0]}</div>
          <span className="text-link">
            View architecture <Icon name="arrow" size={14}/>
          </span>
        </Link>))}
    </div>);
}
function Home() {
    useEffect(() => {
        document.title =
            "Archbase — Architecture that AI coding agents can actually follow";
    }, []);
    const steps = [
        ["Install the CLI", INSTALL],
        ["Consult the catalog", "arc rules list"],
        ["Add a pattern", "arc add next/page@1234 ."],
        ["Resolve a path", "arc resolve src/pages/Home.tsx"],
        [
            "Export agent rules",
            "arc rules add architecture/next-modular@1 --format agents",
        ],
        ["Connect optional tools", "arc mcp serve --project-root ."],
    ];
    return (<main id="main" className="home">
      <section className="hero">
        <div className="hero-grid" aria-hidden="true"/>
        <div className="hero-inner">
          <img className="hero-original-logo" src="/assets/archbase-mark.svg" alt="Archbase architectural ruler and code brace" width="140" height="144"/>
          <div className="eyebrow">
            <span />
            Open source architecture context
          </div>
          <h1>
            Architecture that AI coding agents can <em>actually follow.</em>
          </h1>
          <p>
            Share reusable code patterns and architecture rules with developers
            and AI coding agents. Keep local conventions explicit, resolve the
            right pattern for each directory, and expose project architecture
            through read-only MCP tools.
          </p>
          <div className="hero-actions">
            <Link href="/docs/quick-start" className="button primary">
              Get started <Icon name="arrow" size={16}/>
            </Link>
            <Link href="/docs/introduction" className="button">
              Read the documentation
            </Link>
            <a className="button ghost" href={GITHUB} target="_blank" rel="noreferrer">
              <Icon name="github" size={16}/>
              View on GitHub
            </a>
          </div>
          <div className="hero-command">
            <div>
              <span className="command-label">Linux · official channel</span>
              <code>
                <b>$</b> {INSTALL}
              </code>
            </div>
            <CodeCopy text={INSTALL}/>
          </div>
          <div className="trust-line">
            <span>
              <Icon name="check" size={14}/>
              MIT licensed
            </span>
            <span>
              <Icon name="check" size={14}/>
              Local-first
            </span>
            <span>
              <Icon name="check" size={14}/>
              Read-only MCP
            </span>
            <span>
              <Icon name="check" size={14}/>
              No telemetry
            </span>
          </div>
          <ProjectBadges />
        </div>
      </section>
      <section className="home-section concepts-section">
        <div className="section-heading">
          <div>
            <span className="section-index">01 / FOUNDATIONS</span>
            <h2>One source of architecture context.</h2>
          </div>
          <p>
            Archbase turns conventions into explicit references that remain
            useful across editors, agents, and teams.
          </p>
        </div>
        <div className="concept-grid">
          {[
            [
                "Patterns",
                "Focused references for pages, components, controllers, services, and other code types.",
                "01",
            ],
            [
                "Rules",
                "Responsibilities, path organization, layer relationships, constraints, and pattern links.",
                "02",
            ],
            [
                "Scopes",
                "Deterministic directory-level resolution where the most specific valid scope wins.",
                "03",
            ],
            [
                "MCP",
                "Six read-only tools expose catalog and resolved project context over stdio.",
                "04",
            ],
        ].map(([title, body, no]) => (<div className="concept-card" key={title}>
              <span>{no}</span>
              <h3>{title}</h3>
              <p>{body}</p>
              <Link href={`/docs/${title === "MCP" ? "mcp-setup" : "concepts"}`}>
                Learn more <Icon name="arrow" size={14}/>
              </Link>
            </div>))}
        </div>
      </section>
      <section className="home-section flow-section">
        <div className="section-heading">
          <div>
            <span className="section-index">02 / WORKFLOW</span>
            <h2>From install to agent context.</h2>
          </div>
          <p>
            A short, inspectable path. Every command has one job and local
            choices remain local.
          </p>
        </div>
        <div className="steps">
          {steps.map(([title, code], index) => (<div className="step-row" key={title}>
              <span className="step-no">
                {String(index + 1).padStart(2, "0")}
              </span>
              <h3>{title}</h3>
              <code>{code}</code>
              <CodeCopy text={code} iconOnly/>
            </div>))}
        </div>
        <div className="flow-note">
          <Icon name="terminal"/>
          <p>
            <strong>Interactive when you want it.</strong> Run <code>arc</code>{" "}
            for the terminal UI, or use <code>arc --plain</code> for textual
            output.
          </p>
          <Link href="/docs/terminal-ui">
            Terminal UI <Icon name="arrow" size={14}/>
          </Link>
        </div>
      </section>
      <section className="home-section catalog-section">
        <div className="section-heading">
          <div>
            <span className="section-index">03 / PUBLIC CATALOG</span>
            <h2>Nine stack families. Fourteen rules.</h2>
          </div>
          <p>
            Choose the architecture variant that fits. Variants are
            alternatives, never a cumulative checklist.
          </p>
        </div>
        <CatalogGrid />
        <div className="section-action">
          <Link href="/docs/catalog" className="button">
            Browse the full catalog <Icon name="arrow" size={16}/>
          </Link>
        </div>
      </section>
      <section className="home-section skills-teaser">
        <div className="skills-teaser-copy">
          <span className="section-index">04 / SKILLS</span>
          <h2>Guided workflows, grounded in evidence.</h2>
          <p>
            Start free with project orientation, then use specialized audit,
            migration, and review workflows when you need deeper analysis.
          </p>
          <Link href="/skills/" className="button primary">
            Explore skills <Icon name="arrow" size={16}/>
          </Link>
        </div>
        <div className="skill-mini-list">
          <div>
            <span className="available">Published</span>
            <h3>Architecture Starter</h3>
            <p>Free · v0.2.0</p>
          </div>
          <div>
            <span className="available">Published</span>
            <h3>Next.js Architecture Auditor</h3>
            <p>US$9 one-time · v0.2.0</p>
          </div>
          <div>
            <span className="available">Published on Agensi</span>
            <h3>Nine more specialized workflows</h3>
            <p>Auditing, migration, and PR review</p>
          </div>
        </div>
      </section>
      <Footer />
    </main>);
}
function CodeCopy({ text, iconOnly = false, }: {
    text: string;
    iconOnly?: boolean;
}) {
    const [copied, setCopied] = useState(false);
    return (<button className={iconOnly ? "mini-copy" : "command-copy"} aria-label="Copy command" onClick={() => {
            navigator.clipboard.writeText(text);
            setCopied(true);
            setTimeout(() => setCopied(false), 1400);
        }}>
      <Icon name={copied ? "check" : "copy"} size={15}/>
      {!iconOnly && (copied ? "Copied" : "Copy")}
    </button>);
}
function SkillsPage() {
    const [filter, setFilter] = useState("All");
    const filters = ["All", "Published", "Auditor", "Planning & review"];
    const filtered = skills.filter((skill) => filter === "All" ||
        (filter === "Published" && skill.status === "Published") ||
        (filter === "Auditor" && skill.family === "Auditor") ||
        (filter === "Planning & review" &&
            ["Planning", "Review"].includes(skill.family)));
    useEffect(() => {
        document.title = "Skills · Archbase";
    }, []);
    return (<main id="main" className="skills-page">
      <section className="skills-hero">
        <div className="eyebrow">
          <span />
          Archbase skills
        </div>
        <h1>
          Architecture workflows with <em>clear boundaries.</em>
        </h1>
        <p>
          Orientation, auditing, migration planning, and pull request review
          grounded in explicit references and file-level evidence.
        </p>
        <div className="license-strip">
          <div>
            <strong>Open source · MIT</strong>
            <span>CLI, MCP, catalog, patterns, rules, Starter, and docs</span>
          </div>
          <div>
            <strong>Premium</strong>
            <span>Original proprietary workflows and templates</span>
          </div>
        </div>
      </section>
      <section className="skills-directory">
        <div className="directory-head">
          <div>
            <span className="section-index">DIRECTORY / 11 SKILLS</span>
            <h2>Find the right workflow.</h2>
          </div>
          <div className="filters" role="group" aria-label="Filter skills">
            {filters.map((item) => (<button className={filter === item ? "active" : ""} onClick={() => setFilter(item)} key={item}>
                {item}
              </button>))}
          </div>
        </div>
        <div className="skills-grid">
          {filtered.map((skill, index) => (<article className="skill-card" key={skill.slug}>
              <div className="skill-card-head">
                <span className="skill-number">
                  {String(skills.indexOf(skill) + 1).padStart(2, "0")}
                </span>
                <span className={`skill-status ${skill.status === "Published" ? "published" : ""}`}>
                  <i />
                  {skill.status}
                </span>
              </div>
              <div className="skill-family">{skill.family}</div>
              <h3>{skill.name}</h3>
              <p>{skill.summary}</p>
              <div className="skill-meta">
                <span>v{skill.version}</span>
                <strong>{skill.price}</strong>
              </div>
              <div className="skill-actions">
                <Link href={`/docs/${skill.slug}`}>
                  Read documentation <Icon name="arrow" size={14}/>
                </Link>
                {skill.url ? (<a className="get-skill" href={skill.url} target="_blank" rel="noreferrer">
                    {skill.price === "Free"
                    ? "Get free skill"
                    : "View on Agensi"}{" "}
                    <Icon name="external" size={13}/>
                  </a>) : (<span className="review-label">Not yet available</span>)}
              </div>
            </article>))}
        </div>
        <div className="evaluation-note">
          <span>Evaluation context</span>
          <p>
            Starter, Next.js Auditor, and each of the seven new auditors use 30
            local scenarios per product. Migration Planner uses 6; PR
            Architecture Guard uses 12. Evaluations use Codex CLI and supplied
            snapshots—they are not framework builds, certification for other
            agents, or proof of Agensi delivery behavior.
          </p>
        </div>
      </section>
      <Footer />
    </main>);
}
// Keep the original static badge in view-source and place that same node in
// the rendered footer. Restore it when navigating away from a page with one.
function attachToolIndexBadge(host: HTMLDivElement) {
    const badge = document.getElementById("toolindex-badge");
    const fallback = document.getElementById("toolindex-footer");
    if (!badge || !fallback) return;
    host.append(badge);
    return () => { fallback.append(badge); };
}
function Footer() {
    return (<footer>
      <div className="footer-brand">
        <Link href="/" className="brand">
          <Logo />
          <span>Archbase</span>
        </Link>
        <p>Architecture that AI coding agents can actually follow.</p>
        <div className="footer-listings" ref={attachToolIndexBadge} />
      </div>
      <div>
        <strong>Documentation</strong>
        <Link href="/docs/quick-start">Quick start</Link>
        <Link href="/docs/catalog">Catalog</Link>
        <Link href="/docs/mcp-setup">MCP setup</Link>
      </div>
      <div>
        <strong>Project</strong>
        <a href={GITHUB} target="_blank" rel="noreferrer">
          GitHub
        </a>
        <a href={`${GITHUB}/releases/latest`} target="_blank" rel="noreferrer">
          Releases
        </a>
        <Link href="/docs/privacy">Privacy</Link>
      </div>
      <div className="footer-bottom">
        <span>Open source under the MIT License.</span>
        <span>Built without analytics or tracking.</span>
      </div>
    </footer>);
}
function ProjectBadges() {
    return (<div className="project-badges" aria-label="Project community links">
      <a href="https://hub.xerj.org/contributors/EnzoCaetano015" target="_blank" rel="noreferrer">
        <img src="https://hub.xerj.org/contributors/EnzoCaetano015/badge.svg" alt="XERJ Corpus Hub contributor"/>
      </a>
      <a href="https://www.agensi.io/creators/enzo-rodrigues" target="_blank" rel="noreferrer">
        <img src="https://img.shields.io/badge/Agensi-11%20published%20skills-147d5a" alt="Archbase skills on Agensi"/>
      </a>
    </div>);
}
function NotFound() {
    return (<main className="not-found">
      <span>404</span>
      <h1>That page is outside the scope.</h1>
      <p>The documentation path you requested does not exist or has moved.</p>
      <Link className="button primary" href="/docs/introduction">
        Open documentation <Icon name="arrow" size={16}/>
      </Link>
    </main>);
}
function App() {
    const [path, setPath] = useState(window.location.pathname);
    const [menuOpen, setMenuOpen] = useState(false);
    const [searchOpen, setSearchOpen] = useState(false);
    const [theme, setThemeState] = useState(() => localStorage.getItem("archbase-theme") ||
        (matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark"));
    const setTheme = (value: string) => {
        setThemeState(value);
        localStorage.setItem("archbase-theme", value);
    };
    useEffect(() => {
        const listener = () => {
            setPath(window.location.pathname);
            setMenuOpen(false);
        };
        window.addEventListener("popstate", listener);
        return () => window.removeEventListener("popstate", listener);
    }, []);
    useEffect(() => {
        document.documentElement.dataset.theme = theme;
    }, [theme]);
    useEffect(() => {
        const listener = (event: KeyboardEvent) => {
            if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
                event.preventDefault();
                setSearchOpen(true);
            }
        };
        window.addEventListener("keydown", listener);
        return () => window.removeEventListener("keydown", listener);
    }, []);
    const docMatch = path.match(/^\/docs\/([^/]+)\/?$/);
    useEffect(() => {
        const slug = path.match(/^\/docs\/([^/]+)\/?$/)?.[1];
        const target = slug && docAliases[slug];
        if (target) {
            window.history.replaceState({}, "", `/docs/${target}`);
            setPath(window.location.pathname);
        }
    }, [path]);
    useEffect(() => {
        const hash = window.location.hash.slice(1);
        if (hash)
            requestAnimationFrame(() => document.getElementById(decodeURIComponent(hash))?.scrollIntoView());
        const page = docMatch && pageMap.get(docMatch[1]);
        const canonical = `https://archbase.caetanodev.com${window.location.pathname}`;
        document
            .querySelector('link[rel="canonical"]')
            ?.setAttribute("href", canonical);
        document
            .querySelector('meta[name="description"]')
            ?.setAttribute("content", page?.description ??
            "Archbase: public architecture patterns, rules, read-only MCP and agent skills.");
    }, [path]);
    const content = path === "/" ? (<Home />) : path === "/skills" || path === "/skills/" ? (<SkillsPage />) : docMatch && pageMap.has(docMatch[1]) ? (<DocsPage slug={docMatch[1]} menuOpen={menuOpen} closeMenu={() => setMenuOpen(false)}/>) : (<NotFound />);
    return (<>
      <a href="#main" className="skip-link">
        Skip to content
      </a>
      <Header onMenu={() => setMenuOpen(true)} onSearch={() => setSearchOpen(true)} theme={theme} setTheme={setTheme}/>
      {content}
      {menuOpen && !docMatch && (<Sidebar open current={undefined} close={() => setMenuOpen(false)}/>)}
      <SearchDialog open={searchOpen} close={() => setSearchOpen(false)}/>
    </>);
}
export default App;
