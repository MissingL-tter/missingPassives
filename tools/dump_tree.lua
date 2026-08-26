-- Exports TreeData/<version>/tree.lua — the passive tree data PoB converted
-- from GGG's JSON — to data/raw/tree_<version>.json (canon encoding, exact
-- floats). sprites.lua and the image assets are view-only and stay behind.
-- Regenerate when the archive updates:
--
--   cd .archive/src && luajit ../../tools/dump_tree.lua [version]
--
-- version defaults to 3_29 (every corpus build's treeVersion).

local version = ... or "3_29"
local canon = dofile("../../tools/canon.lua")

local tree = assert(loadfile("TreeData/" .. version .. "/tree.lua"))()

local out = assert(io.open("../../data/raw/tree_" .. version .. ".json", "w"))
out:write(canon.encodeExact(tree))
out:write("\n")
out:close()

local nodes = 0
for _ in pairs(tree.nodes) do
	nodes = nodes + 1
end
print(string.format("tree %s: %d nodes", version, nodes))
