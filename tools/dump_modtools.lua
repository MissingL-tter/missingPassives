-- Dumps modLib's formatting/parsing/comparison behaviour over every parsed
-- modifier of the corpus to test/testdata/modtools_archive.jsonl.
--
-- Run from .archive/src/:  luajit ../../tools/dump_modtools.lua
--
-- One record per corpus line that parses to modifiers:
--   line     the corpus line
--   fmt      modLib.formatMod per mod
--   params   modLib.formatModParams per mod
--   tags     modLib.formatTags per mod (over its tag array)
--   srcfmt   modLib.formatSourceMod per mod, after setSource("GoPort") on a
--            deep copy
--   ptags    canon(modLib.parseTags(tags[i])) — the round-trip
--   psrc     canon(modLib.parseFormattedSourceMod(srcfmt[i]))
--   selfcmp  compareModParams(mod, deep copy of itself) per mod
--   crosscmp compareModParams(first mod, previous record's first mod)
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

local function jsonStringArray(list)
	local parts = {}
	for i, s in ipairs(list) do
		parts[i] = canon.quote(s)
	end
	return "[" .. table.concat(parts, ",") .. "]"
end

local function jsonBoolArray(list)
	local parts = {}
	for i, b in ipairs(list) do
		parts[i] = tostring(b)
	end
	return "[" .. table.concat(parts, ",") .. "]"
end

local out = assert(io.open("../../test/testdata/modtools_archive.jsonl", "w"))
local records, prevFirst = 0, nil
for _, line in ipairs(lines) do
	local modList = modLib.parseMod(line)
	if modList and #modList > 0 then
		local fmt, params, tags, srcfmt, ptags, psrc, selfcmp = {}, {}, {}, {}, {}, {}, {}
		for i, mod in ipairs(modList) do
			fmt[i] = modLib.formatMod(mod)
			params[i] = modLib.formatModParams(mod)
			tags[i] = modLib.formatTags(mod)
			local copy = copyTable(mod)
			modLib.setSource(copy, "GoPort")
			srcfmt[i] = modLib.formatSourceMod(copy)
			ptags[i] = canon.encode(modLib.parseTags(tags[i]))
			psrc[i] = canon.encode(modLib.parseFormattedSourceMod(srcfmt[i]))
			selfcmp[i] = modLib.compareModParams(mod, copyTable(mod))
		end
		local crosscmp = "null"
		if prevFirst then
			crosscmp = tostring(modLib.compareModParams(modList[1], prevFirst))
		end
		prevFirst = modList[1]
		out:write('{"line":', canon.quote(line),
			',"fmt":', jsonStringArray(fmt),
			',"params":', jsonStringArray(params),
			',"tags":', jsonStringArray(tags),
			',"srcfmt":', jsonStringArray(srcfmt),
			',"ptags":', jsonStringArray(ptags),
			',"psrc":', jsonStringArray(psrc),
			',"selfcmp":', jsonBoolArray(selfcmp),
			',"crosscmp":', crosscmp, "}\n")
		records = records + 1
	end
end
out:close()
io.stderr:write(string.format("modtools archive dump: %d records\n", records))
