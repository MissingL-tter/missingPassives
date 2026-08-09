---
description: Compress .claude instruction files via the claudifier subagent.
argument-hint: [path to limit scope, default all of .claude/]
---

Scope: `$ARGUMENTS` if given, else all of `.claude/`.

Gate, before launching anything: `git status --porcelain -- .claude/`. Any output at all
(modified OR untracked) -> do not launch: relay the file list, ask whether to commit, end the
turn. Never commit to clear it. Only empty output proceeds. The agent does not check this
itself - this is the only guard.

Then snapshot mtimes across `.claude/`; the agent reports only the files it believes it
touched.

Launch the `claudifier` subagent (Agent tool, `subagent_type: claudifier`,
`run_in_background: false`). Do not restate its rules in the launch prompt - it reads its own
spec (`GENERATED` files, `recipes/`, no git writes, verification method). Pass scope plus
anything unusual about this run, nothing else.

On return, diff mtimes to confirm nothing out of scope was written. Do not re-run its
`luajit -bl` / `validate.lua` checks - its spec requires both before it reports. Re-run only
if it reports a failure, says it skipped one, or edited `.lua` without mentioning them.

Relay: bytes before -> after per file, the mtime diff, and any fact it reports dropping.
Nothing else - it gives no rationale and you add none. Its pass/skip lines are yours to act
on, not to relay.
