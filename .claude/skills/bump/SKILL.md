---
name: bump
description: Rebase the .archive submodule onto a newer PathOfBuilding release and re-verify every differential.
disable-model-invocation: true
---

`.archive` is the differential reference: every parity test compares Go output against the Lua in
it. Bumping it rewrites that reference, so fixture drift is expected — the drift report is this
skill's output, not a failure to hide.

Our commits sit on top of a PathOfBuilding release tag. The bump rebases them onto a newer tag,
keeping them on top.

## Preconditions

Report and stop if any fails. Never `git stash`, reset or force anything to get past one.

- cwd repo on branch `go`, `git status` clean
- `git -C .archive status` clean
- `.archive` remotes: `origin` = our archive repo, `upstream` =
  `https://github.com/PathOfBuildingCommunity/PathOfBuilding.git` (add if missing)

## 1. Target

    git -C .archive fetch upstream --tags
    git -C .archive describe --tags --abbrev=0 --match 'v*'    # BASE, e.g. v2.67.2
    git -C .archive log --oneline -1 upstream/master           # target

`upstream/master` is PoB's release branch. `upstream/dev` is never a bump target.
If `upstream/master` is already the merge-base with `master`, report "already current" and stop.

## 2. Tag the outgoing tip

Every past `go` commit pins an `.archive` SHA that the rebase orphans. The tag keeps those
resolvable.

    git -C .archive tag archive-BASE master
    git -C .archive push origin archive-BASE

Tag name is `archive-` + BASE, e.g. `archive-v2.67.2`. Ask before pushing.

## 3. Rebase

    git -C .archive rebase upstream/master

Conflicts land in PathOfBuilding's Lua. Per conflict:

- our commit's intent wins; adopt upstream's surrounding rewrite around it
- upstream shipped the same fix -> `git rebase --skip`, record which commit was dropped
- unclear -> stop and ask; never guess at PoE mechanics

`git -C .archive rebase --abort` restores the pre-rebase state, as does the step-2 tag.

## 4. Publish

    git -C .archive push --force-with-lease origin master

This rewrites published history. Ask every time; never pass plain `--force`.

`.archive` is the only working copy. The archive repo is not a GitHub fork of PathOfBuilding, so
upstream reaches it only through the `upstream` remote above.

## 5. Re-point the parent

    git add .archive
    git commit -m "Bump .archive to PathOfBuilding <new tag>"

## 6. Regenerate, then report drift

The GGPK is not in git. A league bump needs `.archive/src/Export/ggpk` re-extracted from the new
`Content.ggpk` first, or every artifact regenerates from stale game data.

    go run ./cmd/sourceupdate -treetag <GGG skilltree-export release tag>

That rewrites `data/raw`, then runs `go test ./... -count=1` and the `MP_EXPORT` export
differential. Failures are the drift report.

Read `.claude/documentation/parity.md` before touching any fixture. A difference is either the new
PoB version being correct or a real port regression — fix the port; re-baseline a fixture only with
a stated reason.

Final report: BASE -> new tag, commits rebased, commits dropped and why, tests still failing.
