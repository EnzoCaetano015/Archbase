# Starter evaluation fixtures

These are versioned test inputs, not installable Next.js applications. The three
public scenarios keep evaluation-contract tests independent of the Premium repo.
No dependency installation or project-script execution is needed.

Every fixture file is stored as a typed Markdown document with one additional
`.md` suffix: `app/page.tsx.md` represents `app/page.tsx`, and
`package.json.md` represents `package.json`. The code fence declares the original
file type. This keeps incomplete samples outside normal JSX/TypeScript analysis.

An original Markdown file also gets the suffix: `AGENTS.md.md` represents
`AGENTS.md`. A pattern example named `example.tsx.md` is stored as
`example.tsx.md.md`, with its original Markdown inside an outer Markdown fence.
Fixture policy and adversarial text remain test data, not repository instructions.

`suite.json` evidence paths refer to decoded project files. The loader validates
documents and computes hashes from logical paths and extracted code, not wrappers.
To create real filenames for a local evaluation, run from the OSS root:

```bash
go run ./internal/skillbundle/cmd/skillbundle eval-fixture \
  -suite internal/skillbundle/testdata/starter/suite.json \
  -scenario s13-small-client-island -output /tmp/starter-client-example
```

Choose a new destination outside the versioned fixtures. The command never executes
the extracted project and refuses an existing destination. Use only the selected
project, prompt and skill when evaluating an agent; keep evaluator assertions out
of its input. Contract tests with synthetic records are not behavioral evaluations.
