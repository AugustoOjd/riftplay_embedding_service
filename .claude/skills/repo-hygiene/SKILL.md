---
name: repo-hygiene
description: >-
  Use before any commit or push to check that staged or outgoing files do not
  include junk: compiled Go binaries, test binaries, coverage output, vendor/,
  scraped HTML dumps, database dumps, logs, editor files, or large binaries.
  Invoked by /commit and /push_github; also use when setting up or reviewing
  .gitignore.
---

# Repo hygiene

Keep build output and generated junk out of git history. A large blob, once
committed and pushed, is in every clone forever.

## What to check

The caller says which input applies:

- **Staged changes** (from `/commit`):
  ```bash
  git diff --cached --name-only --diff-filter=A
  git diff --cached --stat
  ```
- **Outgoing range** (from `/push_github`):
  ```bash
  git log --name-only --diff-filter=A --format= <range> | sort -u
  git diff --stat <range>
  ```

Get sizes of added files:

```bash
git diff --cached --name-only --diff-filter=A -z | xargs -0 -r du -k | sort -rn | head -20
```

For a range, check blob sizes in history:

```bash
git rev-list --objects <range> | git cat-file --batch-check='%(objecttype) %(objectsize) %(rest)' \
  | awk '$1=="blob" && $2>1048576 {print $2, $3}' | sort -rn
```

## Junk for this Go project

| Kind | Examples |
|---|---|
| Compiled binaries | `embedding` at repo root or in `cmd/*/`, `bin/`, `dist/`, `*.exe`, any ELF/Mach-O file (`file <path>` says "executable") |
| Test binaries & profiles | `*.test`, `*.prof`, `cpu.out`, `mem.out`, `trace.out` |
| Coverage | `coverage.out`, `coverage.html`, `*.cover`, `*.coverprofile` |
| Dependencies | `vendor/` — only allowed if vendoring was explicitly decided |
| Env & local config | `.env`, `.env.*` (except `.env.example`), `*.local.*` — also a secret-scan concern |
| Scraping output | HTML/JSON dumps outside `testdata/`, `chromedp`/`rod` user-data dirs, screenshots, downloaded browser binaries |
| Data | `*.sql.gz`, `*.dump`, `*.backup`, `*.csv`/`*.parquet` exports |
| Logs | `*.log`, `logs/` |
| Editor / OS | `.idea/`, `.vscode/` (except shared `settings.json` if agreed), `*.swp`, `.DS_Store`, `Thumbs.db` |
| Claude local files | `.claude/settings.local.json` |

**Not junk** — these belong in git:

- `go.mod`, `go.sum`
- `sqlc`-generated Go code (committed, and checked with `sqlc diff`)
- Small HTML/JSON fixtures under `testdata/` (keep each under ~500 KB; trim pages
  to the part the test needs)
- `.env.example` with placeholder values
- `.claude/` commands, skills, agents and `settings.json`

## Deciding

- Any file over **10 MB** → stop.
- **1–10 MB** → ask the user.
- Anything in the junk table → unstage it, whatever its size.
- A file with no extension at the root or in `cmd/*/` → run `file <path>`; a Go
  binary is the most common accident (`go build` in a `cmd/` dir without `-o`).

When something is unstaged, add a matching `.gitignore` rule **in the same
commit**, or the next `git add .` brings it back. A baseline for this repo:

```gitignore
# Binaries
/embedding
/bin/
/dist/
*.exe
*.test

# Profiling & coverage
*.out
*.prof
coverage.html
*.cover

# Env
.env
.env.*
!.env.example

# Local
*.log
.DS_Store
.idea/
.claude/settings.local.json
```

## Reporting

List each finding with path, size and why it is junk, plus the `.gitignore` rule
you would add. For a range, a finding means rewriting unpushed history — the
user decides. No findings → one line saying what was checked.
