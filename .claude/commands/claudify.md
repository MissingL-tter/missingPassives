---
description: Compress .claude instruction files via the claudifier subagent.
argument-hint: [path to limit scope, default all of .claude/] [-gitskip]
---

Scope: `$ARGUMENTS` if given, else all of `.claude/`. Strip a `-gitskip` flag from the
arguments before treating the rest as scope. Only valid outside-repo scope: the user's
global `E:\env\claude\CLAUDE.md` (standing target the agent spec knows; pass as plain scope,
nothing unusual). Scope-dir = the given file's directory, or `.claude/` when no argument.

Gate before launching, in-repo scope: `git status --porcelain -- .claude/`. Any output
(modified OR untracked) -> do not launch: relay the file list, ask whether to commit, end
turn. Never commit to clear it. Only empty output proceeds. `-gitskip` waives this gate for
the run: proceed on a dirty tree, but first copy the scoped file(s) to the session
scratchpad as a restore point and name that path in the relay. Global-CLAUDE.md scope: no
git gate ever; same scratchpad restore point. The agent checks none of this itself - this is
the only guard.

Then snapshot mtimes across scope-dir; the agent reports only files it believes it touched.

Launch the `claudifier` subagent (Agent tool, `subagent_type: claudifier`,
`run_in_background: false`). Do not restate its rules in the launch prompt - it reads its
own spec (`GENERATED` files, `recipes/`, no git writes, verification method). Pass scope
plus anything unusual about this run, nothing else.

On return, diff mtimes to confirm nothing out of scope was written. Do not re-run its
`luajit -bl` / `validate.lua` checks - its spec requires both before it reports. Re-run only
if it reports a failure, says it skipped one, or edited `.lua` without mentioning them.

Relay: bytes before -> after per file, the mtime diff, any fact it reports dropping. Nothing
else - it gives no rationale and you add none. Its pass/skip lines are yours to act on, not
to relay.
