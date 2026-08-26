-- Exports Data/ModCache.lua — PoB's shipped file of pre-parsed mod lines —
-- to data/raw/modCache.jsonl, one record per line:
--   {"k":<mod line>,"m":<parsed mods, %.17g floats>,"e":<unparsed remainder or null>}
-- The Go parser serves these entries for cached lines instead of parsing,
-- exactly as PoB does (Main.lua L125 preloads the file). Regenerate when
-- the archive updates:
--
--   cd .archive/src && luajit ../../tools/dump_modcache.lua

local canon = dofile("../../tools/canon.lua")

local corpus = {}
assert(loadfile("Data/ModCache.lua"))(corpus)

local keys = {}
for line in pairs(corpus) do
	keys[#keys + 1] = line
end
table.sort(keys)

local out = assert(io.open("../../data/raw/modCache.jsonl", "w"))
for _, line in ipairs(keys) do
	local dat = corpus[line]
	out:write('{"k":', canon.quote(line),
		',"m":', dat[1] and canon.encodeExact(dat[1]) or "null",
		',"e":', dat[2] and canon.quote(dat[2]) or "null", "}\n")
end
out:close()
print(string.format("modCache: %d entries", #keys))
