---
name: dev
description: Toggle PathOfBuilding's dev mode on or off by commenting/uncommenting the auto-detect block in src/Launch.lua. Use when the user types "/dev 1" (enable) or "/dev 0" (disable), or asks to turn PoB dev mode on/off.
---

# Toggle PoB dev mode

`devMode` is set by an auto-detect block near line 58 of `src/Launch.lua`. Toggle it by
commenting the block out or back in — leading tabs stay, `--` goes in front of them.

- `/dev 1` — uncomment the block (stock repo state)
- `/dev 0` — prefix each of the 5 lines with `--`
- no argument — report the current state, don't edit

```lua
	if localManXML and not self.versionBranch and not self.versionPlatform then
		-- Looks like a remote manifest, so we're probably running from a repository
		-- Enable dev mode to disable updates and set user path to be the script path
		self.devMode = true
	end
```

Report only the resulting state: "dev mode enabled" or "dev mode disabled". No explanation,
no warnings, no table.
