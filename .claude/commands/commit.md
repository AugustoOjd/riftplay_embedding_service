---
description: Stage everything, scan for exposed secrets and junk, and commit locally — never pushes
argument-hint: "[mensaje corto, opcional]"
allowed-tools: Bash(git:*), Read, Grep, Glob, Skill
---

Stage and commit **locally**. Message: **$ARGUMENTS** (derive one when empty — see
below). This command never pushes; publishing is `/push_github`.

## 1. Check the ground first

```bash
git status --porcelain
git branch --show-current
```

Stop and say why, without staging anything, if there is nothing to commit or HEAD
is detached. No remote is needed: a local commit does not touch one.

**Commit on the branch that is checked out — never create, switch or rename a
branch** (no `git switch -c`, `git checkout -b`, `git branch -m`), not even when
the current branch looks already merged or wrong. If it does look wrong (e.g.
`main`, or a branch already merged), say so and ask; which branch to use is the
user's decision.

## 2. Stage, then look at what got staged

```bash
git add .
git diff --cached --stat
git diff --cached --name-only --diff-filter=A
git diff --cached --name-only --diff-filter=D
```

`git add .` sweeps the whole directory, so the other commands are the point of
this step, not decoration. Stop and ask about anything unrelated to the work just
done — including deletions nobody mentioned.

## 3. Two checks, both gating

Run both against the staged changes before committing:

- The **`secret-scan`** skill — credentials.
- The **`repo-hygiene`** skill — compiled Go binaries (e.g. `embedding`,
  `bin/`, `dist/`), `*.test` binaries, coverage output (`coverage.out`,
  `*.cover`), `vendor/` (unless vendoring was decided), `.env` files, scraped HTML
  dumps outside `testdata/`, database dumps, logs, and other large binaries.

Local is not a reason to relax them. A commit is history: once it is pushed, a key
is compromised and a large blob is in every clone forever, and `/push_github` only
catches it again if nobody pushes by hand. Staged, the fix is one command; committed,
it is a history rewrite. Never wave a match through as "probably a placeholder" or
"probably small" without looking.

**A secret finding stops everything.** A hygiene finding over 10 MB stops too;
between 1 and 10 MB, ask. When something is unstaged for either reason, add the
matching `.gitignore` rule in the same commit — otherwise the next `git add .`
brings it straight back.

## 4. Commit

Use `$ARGUMENTS` verbatim as the subject when given. When empty, write one from
what actually changed, taking the branch name as context (`init_riftplay_project`
→ the initial setup of the project). One line, imperative, under 72 characters,
saying what changed rather than which files moved. Spanish, like the rest of this
repository's prose.

End the message with the co-author trailer, passed with a heredoc so it keeps its
own line:

```bash
git commit -m "$(cat <<'EOF'
<asunto>

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
)"
```

## 5. Report

The commit's short hash and subject, the branch, and how many commits are now ahead
of the upstream (`git status -sb`) — those are what `/push_github` would publish. If
a check stopped the commit, say what remains staged so nothing is left half-done.
