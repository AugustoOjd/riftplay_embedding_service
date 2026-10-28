---
name: secret-scan
description: >-
  Use before any commit or push to scan staged changes or an outgoing commit range
  for exposed credentials: database URLs with passwords, AWS keys, LLM API keys,
  tokens, private keys, and .env files. Invoked by /commit and /push_github; also
  use when asked whether a diff or file leaks a secret.
---

# Secret scan

Find credentials before they enter git history. A secret that reaches a pushed
commit is compromised: it must be **rotated**, not just deleted.

## What to scan

The caller says which input applies:

- **Staged changes** (from `/commit`): `git diff --cached --no-color`
- **Outgoing range** (from `/push_github`): `git log -p --no-color <range>` — every
  commit, not the net diff. A key added in one commit and removed in the next is
  still in the history being pushed.

Only **added lines** (`+`) can leak something new. Still, read file names in the
diff too: a new `.env` file is a finding even if its contents look harmless.

## What counts as a secret in this project

| Kind | Pattern to look for |
|---|---|
| Postgres URL with password | `postgres(ql)?://[^:\s]+:[^@\s]+@` — Railway `DATABASE_URL` values, including `*.railway.internal` and `*.proxy.rlwy.net` hosts |
| AWS access key ID | `\b(AKIA\|ASIA)[0-9A-Z]{16}\b` |
| AWS secret key | 40-char base64-ish value next to `aws_secret_access_key` / `AWS_SECRET_ACCESS_KEY` |
| Anthropic key | `sk-ant-[A-Za-z0-9_-]{20,}` |
| OpenAI key | `sk-(proj-)?[A-Za-z0-9_-]{20,}` |
| Google / Gemini key | `AIza[0-9A-Za-z_-]{35}` |
| GitHub token | `gh[pousr]_[A-Za-z0-9]{36,}`, `github_pat_[A-Za-z0-9_]{50,}` |
| Railway token | value assigned to `RAILWAY_TOKEN` / `RAILWAY_API_TOKEN` |
| Private key | `-----BEGIN [A-Z ]*PRIVATE KEY-----` |
| JWT | `eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}` |
| Generic | a literal assigned to a name containing `password`, `passwd`, `secret`, `token`, `api_key`, `apikey`, `private_key`, `dsn` |

Files that are findings by their name alone: `.env`, `.env.*` (except
`.env.example` / `.env.sample`), `*.pem`, `*.key`, `*.p12`, `id_rsa*`,
`credentials`, `.aws/`, `*.tfstate`.

Useful first pass (then read every hit in context):

```bash
git diff --cached --no-color | grep -E '^\+' | grep -nEi \
  'postgres(ql)?://[^:[:space:]]+:[^@[:space:]]+@|(AKIA|ASIA)[0-9A-Z]{16}|sk-ant-|sk-(proj-)?[A-Za-z0-9_-]{20}|AIza|gh[pousr]_|github_pat_|BEGIN [A-Z ]*PRIVATE KEY|password|secret|token|api_?key'
```

## Go-specific places secrets hide

- String constants or `var` defaults: `const dsn = "postgres://..."`,
  `os.Getenv("X")` with a hardcoded fallback: `cmp.Or(os.Getenv("KEY"), "sk-...")`.
- Struct tag defaults in config libraries: `envDefault:"..."`.
- Test files and `testdata/` — a real key in a test is still a real key. Tests
  must use obvious fakes (`sk-ant-test`, `postgres://user:pass@localhost/test`).
- Recorded HTTP fixtures / cassettes: `Authorization:` headers, `x-api-key:`, cookies.
- `railway.json`, `Dockerfile` (`ENV`, `ARG` with defaults), `docker-compose.yml`,
  GitHub Actions workflows.
- Log lines or error messages that print the full DSN or request headers.

## Deciding

Not a finding — but check before dismissing:

- `os.Getenv("DATABASE_URL")` or `${{Postgres.DATABASE_URL}}` — references, not values.
- `postgres://postgres:postgres@localhost:5432/...` in test/dev config — local only,
  acceptable if clearly local (`localhost`, `127.0.0.1`, a docker service name).
- Placeholders like `<your-key>`, `xxx`, `changeme`, `sk-ant-...` with literal dots.

**Never dismiss a match as "probably a placeholder" without reading it.** If a
value is high-entropy and you cannot tell, it is a finding.

## Reporting

For each finding: file, line, kind, and the value **redacted** (first 4 chars +
`…`). Never print a full secret back in the conversation.

A finding **stops** the commit/push. Recommend, in order:

1. Unstage / remove the value; read it from an environment variable instead.
2. Add the file pattern to `.gitignore` if it is a file (`.env`, key files).
3. If it is already in a commit: the fix is a history rewrite (`--amend`,
   `rebase`) — the user decides.
4. If it was ever pushed or shared: **rotate the credential**. Removing it from
   history does not un-leak it.

No findings → say so in one line, naming what was scanned (staged index or range).
