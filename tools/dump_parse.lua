-- Dumps the reference parser's output for every known mod line to
-- test/testdata/parse_archive.jsonl, one JSON record per line:
--   {"line":<input>,"mods":<canonical>,"extra":<remainder or null>}
--
-- Run from .archive/src/:  luajit ../../tools/dump_parse.lua
--
-- The corpus is the key set of Data/ModCache.lua — every modifier line the
-- application has ever parsed. Results are recomputed by the live parser, not
-- read from the cache, so the archive dump reflects the parser as it stands.
package.cpath = "../runtime/?.dll;" .. package.cpath
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
dofile("HeadlessWrapper.lua")

local canon = dofile("../../tools/canon.lua")

local corpus = {}
LoadModule("Data/ModCache", corpus)

local lines = {}
for line in pairs(corpus) do
	lines[#lines + 1] = line
end
table.sort(lines)

local out = assert(io.open("../../test/testdata/parse_archive.jsonl", "w"))
local produced, empty, none = 0, 0, 0
for _, line in ipairs(lines) do
	local modList, extra = modLib.parseMod(line)
	if modList and #modList > 0 then
		produced = produced + 1
	elseif modList then
		empty = empty + 1
	else
		none = none + 1
	end
	out:write('{"line":', canon.quote(line),
		',"mods":', modList and canon.encode(modList) or "null",
		',"extra":', extra and canon.quote(extra) or "null", "}\n")
end
out:close()

io.stderr:write(string.format(
	"corpus %d lines: %d produce mods, %d parse to an empty list, %d are not understood\n",
	#lines, produced, empty, none))
