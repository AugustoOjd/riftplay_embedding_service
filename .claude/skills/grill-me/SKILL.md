---
name: grill-me
description: >-
  Interview the user relentlessly about a plan or design until reaching shared
  understanding, resolving each branch of the decision tree. Use when the user
  wants to stress-test a plan, get grilled on their design, pair-program a
  decision before coding, or mentions "grill me".
argument-hint: "[tema, plan o feature a discutir]"
---

# Grill me

We are pair programming on **$ARGUMENTS**. Interview the user relentlessly
about every aspect of the plan until we share one understanding. Walk down
each branch of the design tree, resolving dependencies between decisions
one by one. The user decides; you spot what hasn't been decided yet and
recommend.

Talk in the user's language (Spanish, unless they switch).

## 1. Get the ground before asking

If a question can be answered by exploring the codebase or files, explore
them yourself instead of asking. Before the first question, read what's
relevant: [RAG_DESIGN.md](../../../RAG_DESIGN.md) (decisions already taken and
the ones marked **abiertas**), `CLAUDE.md`, and the code the topic touches.
If `$ARGUMENTS` is empty, the first question is what we're working on.

Then say in two or three lines what you understood and what is still open.

## 2. How to ask questions

Use the **AskUserQuestion** tool for every question. Never pose questions as
plain text — always the multiple-choice popup, so the user can pick an
answer or type a custom one in "Other".

- **One question at a time.** Wait for the answer before the next one.
- **2–4 concrete options** representing the most likely directions. Generic
  "Sí" / "No" only when the question is genuinely binary.
- **Always recommend.** Put your pick first with "(Recomendado)" at the end
  of its label. Each option's description gives the trade-off, tied to this
  project (Go stdlib, the stack table in `CLAUDE.md`, Railway memory limits,
  the `token_usages` budget, decisions in `RAG_DESIGN.md`) rather than
  generic best practice.
- Use `preview` when comparing concrete artifacts: a struct, a table
  schema, a JSON contract, a function signature.
- Prefer concrete questions (a type, a column, an error case, an endpoint
  contract) over abstract ones. Ask about unhappy paths too: failures,
  retries, idempotency, empty input, concurrency, logging, tests.

## 3. Finish

1. Give a concise summary of all decisions made.
2. Offer to record them in `RAG_DESIGN.md` (move items out of **abiertas**,
   add new ones). Edit it only if the user says yes.
3. Propose the smallest next step to implement and wait for the go-ahead.

## Rules

- Don't write code, add packages, tables, endpoints or dependencies during
  the interview — `CLAUDE.md` says the domain is agreed first.
- Don't re-ask what was already decided in this session or the design doc,
  unless new information makes it worth reopening — and say why.
