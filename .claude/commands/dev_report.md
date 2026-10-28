---
description: Write or update today's development report in developments_reports/
allowed-tools: Bash(date:*), Bash(git log:*), Bash(git status:*), Bash(git diff:*), Bash(ls:*), Read, Write, Edit, Glob
---

Write the development report for today into
`developments_reports/<YYYY-MM-DD>.md`, taking the date from `date +%F` — never
from memory or from the conversation.

**Write the report in Spanish.** It is a work log for the person who owns this
repository, not documentation for an agent.

## What this file is

**A final report written before the daily push, not a running log.** It is not
updated live as work happens, and it is not a transcript of the session. It is
what someone reads to know where the project stood at the end of a day.

*Realizado* and *Decisiones* may be filled in alongside the code, as things get
done and decided — that part does not have to wait. *Pendientes* is settled at
the end, because an item is only open once the day is over.

## Gather before writing

1. **This conversation.** What was actually done, decided, and discovered. This
   is the main source — a git diff shows what changed, not why.
2. `git status --porcelain` and `git log --oneline` since the last report, to
   catch changes the conversation did not mention.
3. **The previous report** (the most recent file in `developments_reports/`).
   Every open item in it must be accounted for in this one: still open, done
   today, or dropped — and if dropped, why. A pending item that quietly vanishes
   between reports is the failure mode this file exists to prevent.

If today's file already exists, **update it** rather than starting a new one —
merge the new work into the existing sections instead of appending a second
report to the same day.

## Structure

Three sections. No others — do not add *Hallazgos*, *Verificado* or a summary
section; that content belongs inside the three below.

```markdown
# YYYY-MM-DD

<Two or three sentences: what this day was about.>

## Realizado

<Grouped by area, not chronologically. Link files as [ruta](ruta). Say what
changed and why it changed — a reader six months from now has the diff already.

This also carries what was found and what was proven:

- Bugs, wrong assumptions and things that turned out not to be true, including
  the ones found and fixed the same day — those are the most useful and the
  easiest to forget.
- What was actually run and passed, with the numbers.

What was NOT verified does not go here. An unbuilt image or an unexercised
refactor is an open item: it goes in *Pendientes*.>

## Decisiones

<Only decisions with a trade-off someone could reasonably have made otherwise.
State the alternative that was rejected and the reason. Skip this section when
the day had none.>

## Pendientes

| # | Pendiente | Tipo | Desde |
|---|---|---|---|

<Type is one of: bloqueante, deuda, mejora. "Desde" is the date it first
appeared, carried across reports so an item that keeps slipping is visible.
Order by type, blockers first.

Then, below the table, the items closed today with the reason each one closed —
so a number that disappears from the table can still be traced.>
```

Drop a section that would be empty rather than writing "ninguno" under it — an
empty *Pendientes* is worth stating, the rest is not.

## Tone

Report what happened, including what failed or was left undone. A report that
only lists wins is worthless as a record. Do not pad: if the day was one bug
fix, the report is short.

Finish by telling the user the path of the file and the number of open items,
carried-forward ones separated from the new ones.
