---
description: Commit pending work, scan every outgoing commit for secrets and junk, run the checks CI runs, and push the current branch to GitHub
argument-hint: "[mensaje corto, opcional]"
allowed-tools: Bash(git:*), Bash(gofmt -l:*), Bash(go vet:*), Bash(go build:*), Bash(go test:*), Bash(go mod tidy -diff), Bash(golangci-lint run:*), Bash(sqlc diff), Bash(sqlc vet), Bash(govulncheck:*), Read, Grep, Glob, Skill
---

Publish the current branch to `origin` on GitHub. Message for pending work:
**$ARGUMENTS**. A push is public and cannot be taken back, so every step below
exists to stop before it, not to explain afterwards.

## 1. Check the ground first

```bash
git status -sb
git branch --show-current
git remote get-url origin
```

Stop and say why if HEAD is detached, there is no `origin`, or `origin` is not a
GitHub URL (then this command is the wrong one — say where it points).

**Push the branch that is checked out, to the remote branch of the same name —
never create, switch or rename a branch** (no `git switch -c`, `git checkout -b`,
`git branch -m`, no pushing to a differently named remote branch), not even when
the current branch looks already merged or wrong. If it does look wrong, say so
and ask; which branch to use is the user's decision.

## 2. Commit pending work, with /commit's rules

If `git status --porcelain` shows changes, read
[commit.md](commit.md) and follow its steps 2–4 exactly, with `$ARGUMENTS` as the
message: stage, review what got staged, both gating checks, commit. Do not restate
or shortcut them here — that file is the single source of those rules.

If a check stops the commit, stop the push too.

## 3. Work out what would be published

```bash
git fetch origin
git rev-parse --abbrev-ref --symbolic-full-name @{u}   # fails when there is no upstream
```

- **With an upstream**, the outgoing range is `@{u}..HEAD`.
- **Without one** (a new branch), it is `origin/main..HEAD` — everything the remote
  does not already have.

```bash
git log --oneline <range>
```

Nothing in the range → stop: there is nothing to push.

**Behind the upstream** (`git status -sb` shows `behind`) → stop and say so. Pulling
or rebasing is the user's decision, and a rejected push is not solved with `--force`.

## 4. Scan every outgoing commit, not only the final state

Run the **`secret-scan`** and **`repo-hygiene`** skills against the outgoing range
instead of the staged index:

```bash
git log -p --no-color <range>                       # secret-scan reads this
git diff --stat <range>
git log --name-only --diff-filter=A --format= <range> | sort -u   # repo-hygiene
```

Scan `git log -p`, **not** the net `git diff <range>`: a key added in one commit and
removed in the next is invisible in the net diff but still in the history being
pushed. Commits made by hand never went through `/commit`, so this scan is the only
one they get.

**A finding stops the push.** The commits are still local, so nothing is
compromised yet — but fixing it means rewriting unpushed history (`--amend`,
`rebase`), which is the user's call. Explain the options; do not rewrite on your own.

## 5. Run what CI runs

```bash
gofmt -l .                  # must print nothing
go vet ./...
go build ./...
go test ./... -race -count=1
go mod tidy -diff           # read-only: fails if go.mod/go.sum are out of date
```

If a `.golangci.yml` exists, also run `golangci-lint run`.

If the range touches `go.mod` or `go.sum`, also run `govulncheck ./...`. It only
reports vulnerabilities in code paths the module actually calls, so any finding
is real: stop and ask. If `govulncheck` is not installed, say so rather than
skipping silently.

If the range touches `.sql` query or schema files, or `sqlc.yaml`, also run
`sqlc diff` (read-only: fails when the generated Go code was not regenerated) and
`sqlc vet`.

Integration tests (`-tags integration`), if any, are not part of this step: they
only run when the user asks. **Never run migrations here**, and never point any
command at the Railway database: it holds real data.

Pushing a red build wastes a CI run and hides the next real failure. A failure here
**stops and asks**: the user may push knowingly (a failure already tracked in
`developments_reports/`), but that is their decision, stated in this turn.

## 6. Push

- **On `main` or `master`, ask first**, even when everything above passed. Pushing
  straight to the default branch is a decision the user makes, not a default.
- Never `--force` or `--force-with-lease` unless the user asks for it in this same
  turn.

```bash
git push -u origin "$(git branch --show-current)"
```

`-u` is harmless when the upstream already exists and sets it when it does not.

## 7. Report

- The pushed commits (short hash and subject) and the branch.
- Whether the push created a new remote branch, and the pull-request URL GitHub
  prints for it. Do not open the PR unless asked.
- What the scans and checks found, including anything the user chose to push
  anyway.
- If you stopped, what state is left: what is committed but unpushed, and what is
  still staged.
