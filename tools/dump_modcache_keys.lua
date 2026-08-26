-- Emits the key set of Data/ModCache.lua to data/raw/modCacheKeys.json
-- (sorted JSON array). The shipped cache's VALUES are the fresh parse
-- round-tripped through %.14g text (Main:SaveModCache writeLuaTable), and
-- the parse differential (test/parse_test.go over this same key set at
-- %.14g) proves the shipped values match the ported parser; so the Go side
-- reproduces cache semantics as: fresh parse + %.14g quantization for
-- exactly these keys. Lines outside the set parse fresh at full precision,
-- as they do in PoB.
--
-- Run from .archive/src/:  luajit ../../tools/dump_modcache_keys.lua

local canon = dofile("../../tools/canon.lua")

local corpus = {}
assert(loadfile("Data/ModCache.lua"))(corpus)

local keys = {}
for line in pairs(corpus) do
	keys[#keys + 1] = line
end
table.sort(keys)

local out = assert(io.open("../../data/raw/modCacheKeys.json", "w"))
out:write("[")
for i, line in ipairs(keys) do
	if i > 1 then
		out:write(",\n")
	else
		out:write("\n")
	end
	out:write(canon.quote(line))
end
out:write("\n]\n")
out:close()
print(string.format("modCacheKeys: %d keys", #keys))
