---
name: dev
description: Toggle PathOfBuilding dev mode. Use for "/dev 1", "/dev 0", or requests to turn PoB dev mode on/off.
---

`self.devMode` in `.archive/src/Launch.lua` (~line 21) is the only assignment; edit it in place.

- `/dev 1` → `self.devMode = true`
- `/dev 0` → `self.devMode = false`
- no arg → report current value, don't edit

Reply exactly `dev mode enabled` or `dev mode disabled`. Nothing else.
