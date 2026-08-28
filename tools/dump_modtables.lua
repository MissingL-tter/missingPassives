-- Dumps every entry of every ModParser pattern table, canonically encoded, to
-- test/testdata/tables_archive.jsonl:
--   {"table":<name>,"key":<pattern>,"value":<canonical>}
--
-- Run from .archive/src/:  luajit ../../tools/dump_modtables.lua
--
-- The differential parse test only reaches entries some corpus line matches;
-- this dump lets the Go side verify every DATA entry byte for byte, and that
-- both sides agree on which entries are closures ({"__fn":true}).
package.cpath = "../runtime/?.dll;" .. package.cpath
package.path = "../runtime/lua/?.lua;../runtime/lua/?/init.lua;" .. package.path
dofile("HeadlessWrapper.lua")

local canon = dofile("../../tools/canon.lua")

-- Load a parser copy whose trailing return also hands back its tables.
local src = assert(io.open("Modules/ModParser.lua", "r"))
local text = src:read("*a")
src:close()
local names = {
	"formList", "modNameList", "modFlagList", "preFlagList", "modTagList",
	"specialModList", "unsupportedModList", "suffixTypes", "dmgTypes", "penTypes",
	"resourceTypes", "regenTypes", "degenTypes", "costTypes", "baseCostTypes",
	"flagTypes", "skillNameList", "preSkillNameList", "jewelFuncList",
	"clusterJewelSkills",
}
local fields = {}
for _, name in ipairs(names) do
	fields[#fields + 1] = name .. " = " .. name
end
text = text:gsub("end, cache%s*$", "end, cache, { " .. table.concat(fields, ", ") .. " }")
local _, _, tables = assert(loadstring(text, "@ModParser.lua"))(launch)

local out = assert(io.open("../../test/testdata/tables_archive.jsonl", "w"))
local total = 0
for _, name in ipairs(names) do
	local keys = {}
	for k in pairs(tables[name]) do
		keys[#keys + 1] = k
	end
	table.sort(keys)
	for _, k in ipairs(keys) do
		out:write('{"table":', canon.quote(name),
			',"key":', canon.quote(k),
			',"value":', canon.quote(canon.encode(tables[name][k])), "}\n")
		total = total + 1
	end
end
out:close()
io.stderr:write(string.format("dumped %d entries across %d tables\n", total, #names))
