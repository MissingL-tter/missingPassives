-- Exports the committed Data/TattooPassives.lua — the tattoo override
-- nodes PoB actually loads at runtime — to data/raw/tattooOverrides.json.
-- (data/raw/tattooPassives.json is the GGPK export pipeline's document and
-- differs from this file in shape and content; the spec port needs the
-- runtime one.) Regenerate when the archive updates:
--
--   cd .archive/src && luajit ../../tools/dump_tattoo.lua

local canon = dofile("../../tools/canon.lua")

local tattoo = assert(loadfile("Data/TattooPassives.lua"))()

local out = assert(io.open("../../data/raw/tattooOverrides.json", "w"))
out:write(canon.encodeExact(tattoo))
out:write("\n")
out:close()

local count = 0
for _ in pairs(tattoo.nodes) do
	count = count + 1
end
print(string.format("tattooOverrides: %d nodes", count))
